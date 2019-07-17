package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	ossignal "os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"

	"github.com/google/uuid"
	"github.com/jroimartin/gocui"
	"github.com/peterbourgon/ff"
	"github.com/pkg/errors"

	status "github.com/status-im/status-protocol-go"
	"github.com/status-im/status-protocol-go/sqlite"

	"github.com/status-im/status-console-client/internal/gethservice"
	migrations "github.com/status-im/status-console-client/internal/sqlite"
)

var g *gocui.Gui

var (
	fs       = flag.NewFlagSet("status-term-client", flag.ExitOnError)
	logLevel = fs.String("log-level", "INFO", "log level")

	keyHex = fs.String("keyhex", "", "pass a private key in hex")
	noUI   = fs.Bool("no-ui", false, "disable UI")

	// flags acting like commands
	createKeyPair = fs.Bool("create-key-pair", false, "creates and prints a key pair instead of running")
	addChat       = fs.String("add-chat", "", "add chat using format: type,name[,public-key] where type can be 'private' or 'public' and 'public-key' is required for 'private' type")

	// flags for in-proc node
	dataDir        = fs.String("data-dir", filepath.Join(os.TempDir(), "status-term-client"), "data directory for Ethereum node")
	installationID = fs.String("installation-id", uuid.New().String(), "the installationID to be used")
	noNamespace    = fs.Bool("no-namespace", false, "disable data dir namespacing with public key")
	fleet          = fs.String("fleet", params.FleetBeta, fmt.Sprintf("Status nodes cluster to connect to: %s", []string{params.FleetBeta, params.FleetStaging}))
	configFile     = fs.String("node-config", "", "a JSON file with node config")
	listenAddr     = fs.String("listen-addr", ":30303", "The address the geth node should be listening to")

	// flags for external node
	providerURI = fs.String("provider", "", "an URI pointing at a provider")
)

func main() {
	if err := ff.Parse(fs, os.Args[1:]); err != nil {
		exitErr(errors.Wrap(err, "failed to parse flags"))
	}

	if *createKeyPair {
		key, err := crypto.GenerateKey()
		if err != nil {
			exitErr(err)
		}
		fmt.Printf("Your private key: %#x\n", crypto.FromECDSA(key))
		os.Exit(0)
	}

	var privateKey *ecdsa.PrivateKey

	if *keyHex != "" {
		k, err := crypto.HexToECDSA(strings.TrimPrefix(*keyHex, "0x"))
		if err != nil {
			exitErr(err)
		}
		privateKey = k
	} else {
		k, err := crypto.GenerateKey()
		if err != nil {
			exitErr(err)
		}
		privateKey = k
		fmt.Printf("Starting with a new private key: %#x\n", crypto.FromECDSA(privateKey))
	}

	// Prefix data directory with a public key.
	// This is required because it's not possible
	// or adviced to share data between different
	// key pairs.
	if !*noNamespace {
		*dataDir = filepath.Join(
			*dataDir,
			hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey)[:20]),
		)
	}

	err := os.MkdirAll(*dataDir, 0755)
	if err != nil {
		exitErr(err)
	} else {
		fmt.Printf("Starting in %s\n", *dataDir)
	}

	// Setup logging by splitting it into a client.log
	// with status-console-client logs and status.log
	// with Status Node logs.
	clientLogPath := filepath.Join(*dataDir, "client.log")
	clientLogFile, err := os.OpenFile(clientLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		exitErr(err)
	}
	log.SetOutput(clientLogFile)

	nodeLogPath := filepath.Join(*dataDir, "status.log")
	err = logutils.OverrideRootLog(true, *logLevel, logutils.FileOptions{Filename: nodeLogPath}, false)
	if err != nil {
		exitErr(fmt.Errorf("failed to override root log: %v", err))
	}

	// Create a database.
	// TODO(adam): currently, we use an address as a db encryption key.
	// It should be configurable.
	dbPath := filepath.Join(*dataDir, "db.sql")
	dbKey := crypto.PubkeyToAddress(privateKey.PublicKey).String()
	db, err := sqlite.Open(dbPath, dbKey, sqlite.MigrationConfig{
		AssetNames: migrations.AssetNames(),
		AssetGetter: func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	})
	if err != nil {
		exitErr(fmt.Errorf("failed to open db: %v", err))
	}
	persistence := newSQLitePersistence(db)

	// Log the current chat info in two places for easy retrieval.
	fmt.Printf("Chat address: %#x\n", crypto.FromECDSAPub(&privateKey.PublicKey))
	log.Printf("chat address: %#x", crypto.FromECDSAPub(&privateKey.PublicKey))

	// Handle add chat.
	if *addChat != "" {
		options := strings.Split(*addChat, ",")

		var c Chat

		if len(options) == 2 && options[0] == "public" {
			c = CreatePublicChat(options[1])
		} else if len(options) == 3 && options[0] == "private" {
			c, err = CreateOneToOneChat(options[1], options[2])
			if err != nil {
				exitErr(err)
			}
		} else {
			exitErr(errors.Errorf("invalid -add-chat value"))
		}

		exists, err := persistence.ChatExist(c)
		if err != nil {
			exitErr(err)
		}
		if !exists {
			if err := persistence.AddChats(c); err != nil {
				exitErr(err)
			}
		}
	}

	chats, err := persistence.Chats()
	if err != nil {
		exitErr(err)
	}

	// initialize protocol
	var messenger *status.Messenger

	if *providerURI != "" {
		messenger, err = createMessengerWithURI(*providerURI)
		if err != nil {
			exitErr(err)
		}
	} else {
		messenger, err = createMessengerInProc(privateKey, chats)
		if err != nil {
			exitErr(err)
		}
	}

	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)

	ossignal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("received signal: %v", sig)
		done <- true
	}()

	log.Printf("starting UI...")

	if !*noUI {
		go func() {
			<-done
			exitErr(errors.New("exit with signal"))
		}()

		if err := setupGUI(privateKey, persistence, messenger); err != nil {
			exitErr(err)
		}

		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			exitErr(err)
		}

		g.Close()
	} else {
		<-done
	}
}

func exitErr(err error) {
	if g != nil {
		g.Close()
	}

	fmt.Println(err)
	os.Exit(1)
}

type keysGetter struct {
	privateKey *ecdsa.PrivateKey
}

func (k keysGetter) PrivateKey() (*ecdsa.PrivateKey, error) {
	return k.privateKey, nil
}

func createMessengerWithURI(uri string) (*status.Messenger, error) {
	_, err := rpc.Dial(*providerURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	// TODO: provide Mail Servers in a different way.
	_, err = generateStatusNodeConfig(*dataDir, *fleet, *listenAddr, *configFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate node config")
	}

	// TODO

	return nil, errors.New("not implemented")
}

func createMessengerInProc(pk *ecdsa.PrivateKey, chats []Chat) (*status.Messenger, error) {
	// collect mail server request signals
	signalsForwarder := newSignalForwarder()
	go signalsForwarder.Start()

	// setup signals handler
	signal.SetDefaultNodeNotificationHandler(
		filterMailTypesHandler(signalsForwarder.in),
	)

	nodeConfig, err := generateStatusNodeConfig(*dataDir, *fleet, *listenAddr, *configFile)
	if err != nil {
		exitErr(errors.Wrap(err, "failed to generate node config"))
	}

	statusNode := node.New()

	protocolGethService := gethservice.New(
		statusNode,
		&keysGetter{privateKey: pk},
	)

	services := []gethnode.ServiceConstructor{
		func(ctx *gethnode.ServiceContext) (gethnode.Service, error) {
			return protocolGethService, nil
		},
	}

	if err := statusNode.Start(nodeConfig, services...); err != nil {
		return nil, errors.Wrap(err, "failed to start node")
	}

	shhService, err := statusNode.WhisperService()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Whisper service")
	}

	var (
		publicChats []string
		publicKeys  []*ecdsa.PublicKey
	)

	for _, chat := range chats {
		if chat.Type == PublicChat {
			publicChats = append(publicChats, chat.PublicName())
		} else if chat.Type == OneToOneChat {
			publicKeys = append(publicKeys, chat.PublicKey())
		}
	}

	messenger, err := status.NewMessenger(
		pk,
		&server{node: statusNode},
		shhService,
		*dataDir,
		"db-key",
		*installationID,
		status.WithChats(publicChats, publicKeys, nil),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Messenger")
	}

	// protocolGethService.SetMessenger(messenger)

	return messenger, nil
}

func setupGUI(privateKey *ecdsa.PrivateKey, persistence *sqlitePersistence, messenger *status.Messenger) error {
	var err error

	// global
	g, err = gocui.NewGui(gocui.Output256)
	if err != nil {
		return err
	}

	// prepare views
	vm := NewViewManager(nil, g)

	notifications := NewNotificationViewController(&ViewController{vm, g, ViewNotification})

	chat := NewChatViewController(
		&ViewController{vm, g, ViewChat},
		privateKey,
		messenger,
		func(err error) {
			_ = notifications.Error("Chat error", fmt.Sprintf("%v", err))
		},
	)

	chats := NewChatsViewController(&ViewController{vm, g, ViewChats}, persistence, messenger)
	if err := chats.LoadAndRefresh(); err != nil {
		return err
	}

	inputMultiplexer := NewInputMultiplexer()
	inputMultiplexer.AddHandler(DefaultMultiplexerPrefix, func(b []byte) error {
		log.Printf("default multiplexer handler")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := chat.Send(ctx, b)
		return err
	})
	inputMultiplexer.AddHandler("/chat", ChatCmdFactory(chats))
	// inputMultiplexer.AddHandler("/request", RequestCmdFactory(chat))

	views := []*View{
		&View{
			Name:       ViewChats,
			Enabled:    true,
			Cursor:     true,
			Highlight:  true,
			SelBgColor: gocui.ColorGreen,
			SelFgColor: gocui.ColorBlack,
			TopLeft:    func(maxX, maxY int) (int, int) { return 0, 0 },
			BottomRight: func(maxX, maxY int) (int, int) {
				return int(math.Floor(float64(maxX) * 0.2)), maxY - 4
			},
			Keybindings: []Binding{
				Binding{
					Key:     gocui.KeyArrowDown,
					Mod:     gocui.ModNone,
					Handler: CursorDownHandler,
				},
				Binding{
					Key:     gocui.KeyArrowUp,
					Mod:     gocui.ModNone,
					Handler: CursorUpHandler,
				},
				Binding{
					Key: gocui.KeyEnter,
					Mod: gocui.ModNone,
					Handler: GetLineHandler(func(idx int, _ string) error {
						selectedChat, ok := chats.ChatByIdx(idx)
						if !ok {
							return errors.New("chat could not be found")
						}

						// We need to call Select asynchronously,
						// otherwise the main thread is blocked
						// and nothing is rendered.
						go func() {
							chat.Select(selectedChat)
						}()

						return nil
					}),
				},
			},
		},
		&View{
			Name:       ViewChat,
			Enabled:    true,
			Cursor:     true,
			Autoscroll: false,
			Highlight:  true,
			Wrap:       true,
			SelBgColor: gocui.ColorGreen,
			SelFgColor: gocui.ColorBlack,
			TopLeft: func(maxX, maxY int) (int, int) {
				return int(math.Ceil(float64(maxX) * 0.2)), 0
			},
			BottomRight: func(maxX, maxY int) (int, int) {
				return maxX - 1, maxY - 4
			},
			Keybindings: []Binding{
				Binding{
					Key:     gocui.KeyArrowDown,
					Mod:     gocui.ModNone,
					Handler: CursorDownHandler,
				},
				Binding{
					Key:     gocui.KeyArrowUp,
					Mod:     gocui.ModNone,
					Handler: CursorUpHandler,
				},
				Binding{
					Key: gocui.KeyHome,
					Mod: gocui.ModNone,
					Handler: func(g *gocui.Gui, v *gocui.View) error {
						return HomeHandler(g, v)
					},
				},
				Binding{
					Key:     gocui.KeyEnd,
					Mod:     gocui.ModNone,
					Handler: EndHandler,
				},
			},
		},
		&View{
			Name: ViewInput,
			Title: fmt.Sprintf(
				"%s (as %#x)",
				ViewInput,
				crypto.FromECDSAPub(&privateKey.PublicKey),
			),
			Enabled:   true,
			Editable:  true,
			Cursor:    true,
			Highlight: true,
			TopLeft: func(maxX, maxY int) (int, int) {
				return 0, maxY - 3
			},
			BottomRight: func(maxX, maxY int) (int, int) {
				return maxX - 1, maxY - 1
			},
			Keybindings: []Binding{
				Binding{
					Key:     gocui.KeyEnter,
					Mod:     gocui.ModNone,
					Handler: inputMultiplexer.BindingHandler,
				},
				Binding{
					Key:     gocui.KeyEnter,
					Mod:     gocui.ModAlt,
					Handler: MoveToNewLineHandler,
				},
			},
		},
		&View{
			Name:      ViewNotification,
			Enabled:   false,
			Editable:  false,
			Cursor:    false,
			Highlight: true,
			TopLeft: func(maxX, maxY int) (int, int) {
				return maxX/2 - 50, maxY / 2
			},
			BottomRight: func(maxX, maxY int) (int, int) {
				return maxX/2 + 50, maxY/2 + 2
			},
			Keybindings: []Binding{
				Binding{
					Key: gocui.KeyEnter,
					Mod: gocui.ModNone,
					Handler: func(g *gocui.Gui, v *gocui.View) error {
						log.Printf("Notification Enter binding")

						if err := vm.DisableView(ViewNotification); err != nil {
							return err
						}

						if err := vm.DeleteView(ViewNotification); err != nil {
							return err
						}

						return nil
					},
				},
			},
		},
	}

	bindings := []Binding{
		Binding{
			Key:     gocui.KeyCtrlC,
			Mod:     gocui.ModNone,
			Handler: QuitHandler,
		},
		Binding{
			Key:     gocui.KeyTab,
			Mod:     gocui.ModNone,
			Handler: NextViewHandler(vm),
		},
	}

	if err := vm.SetViews(views); err != nil {
		return err
	}

	if err := vm.SetGlobalKeybindings(bindings); err != nil {
		return err
	}

	return nil
}
