package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jroimartin/gocui"
	"github.com/peterbourgon/ff"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/adapters"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/gethservice"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
)

var g *gocui.Gui

var (
	fs       = flag.NewFlagSet("status-term-client", flag.ExitOnError)
	logLevel = fs.String("log-level", "INFO", "log level")

	// flags acting like commands
	createKeyPair = fs.Bool("create-key-pair", false, "creates and prints a key pair instead of running")

	// flags for in-proc node
	dataDir    = fs.String("data-dir", filepath.Join(os.TempDir(), "status-term-client"), "data directory for Ethereum node")
	fleet      = fs.String("fleet", params.FleetBeta, fmt.Sprintf("Status nodes cluster to connect to: %s", []string{params.FleetBeta, params.FleetStaging}))
	configFile = fs.String("node-config", "", "a JSON file with node config")
	pfsEnabled = fs.Bool("pfs", false, "enable PFS")

	// flags for external node
	providerURI = fs.String("provider", "", "an URI pointing at a provider")

	keyHex = fs.String("keyhex", "", "pass a private key in hex")
)

func main() {
	log.SetOutput(os.Stderr)

	if err := ff.Parse(fs, os.Args[1:]); err != nil {
		exitErr(errors.Wrap(err, "failed to parse flags"))
	}

	err := logutils.OverrideRootLog(true, *logLevel, logutils.FileOptions{}, false)
	if err != nil {
		log.Fatalf("failed to override root log: %v\n", err)
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

		log.Printf("contact address: %#x", crypto.FromECDSAPub(&privateKey.PublicKey))
	} else {
		exitErr(errors.New("private key is required"))
	}

	// initialize protocol
	var proto protocol.Protocol

	if *providerURI != "" {
		rpc, err := rpc.Dial(*providerURI)
		if err != nil {
			exitErr(errors.Wrap(err, "failed to dial"))
		}

		// TODO: provide Mail Servers in a different way.
		nodeConfig, err := generateStatusNodeConfig(*dataDir, *fleet, *configFile)
		if err != nil {
			exitErr(errors.Wrap(err, "failed to generate node config"))
		}

		proto = adapters.NewWhisperClientAdapter(
			rpc,
			privateKey,
			nodeConfig.ClusterConfig.TrustedMailServers,
		)
	} else {
		// collect mail server request signals
		signalsForwarder := newSignalForwarder()
		defer close(signalsForwarder.in)
		go signalsForwarder.Start()

		// setup signals handler
		signal.SetDefaultNodeNotificationHandler(
			filterMailTypesHandler(printHandler, signalsForwarder.in),
		)

		nodeConfig, err := generateStatusNodeConfig(*dataDir, *fleet, *configFile)
		if err != nil {
			exitErr(errors.Wrap(err, "failed to generate node config"))
		}

		statusNode := node.New()

		protocolGethService := gethservice.New(
			statusNode,
			&keysGetter{privateKey: privateKey},
		)

		services := []gethnode.ServiceConstructor{
			func(ctx *gethnode.ServiceContext) (gethnode.Service, error) {
				return protocolGethService, nil
			},
		}

		if err := statusNode.Start(nodeConfig, services...); err != nil {
			exitErr(errors.Wrap(err, "failed to start node"))
		}

		shhService, err := statusNode.WhisperService()
		if err != nil {
			exitErr(errors.Wrap(err, "failed to get Whisper service"))
		}

		adapter := adapters.NewWhisperServiceAdapter(statusNode, shhService, privateKey)

		protocolGethService.SetProtocol(adapter)

		// TODO: should be removed from StatusNode
		if *pfsEnabled {
			databasesDir := filepath.Join(*dataDir, "databases")

			if err := os.MkdirAll(databasesDir, 0755); err != nil {
				exitErr(errors.Wrap(err, "failed to create databases dir"))
			}

			if err := adapter.InitPFS(databasesDir); err != nil {
				exitErr(errors.Wrap(err, "initialize PFS"))
			}

			log.Printf("PFS has been initialized")
		}

		proto = adapter
	}

	g, err = gocui.NewGui(gocui.Output256)
	if err != nil {
		exitErr(err)
	}
	defer g.Close()

	// prepare views
	vm := NewViewManager(nil, g)

	dbPath := filepath.Join(*dataDir, "db")
	db, err := client.NewDatabase(dbPath)
	if err != nil {
		exitErr(err)
	}
	// TODO: close the database properly

	notifications := NewNotificationViewController(&ViewController{vm, g, ViewNotification})

	messenger := client.NewMessenger(proto, privateKey, db)

	chat := NewChatViewController(
		&ViewController{vm, g, ViewChat},
		privateKey,
		messenger,
		func(err error) {
			_ = notifications.Error("Chat error", fmt.Sprintf("%v", err))
		},
	)

	adambContact, err := client.ContactWithPublicKey("adamb", "0x0493ac727e70ea62c4428caddf4da301ca67b699577988d6a782898acfd813addf79b2a2ca2c411499f2e0a12b7de4d00574cbddb442bec85789aea36b10f46895")
	if err != nil {
		exitErr(err)
	}

	if contacts, err := db.Contacts(); len(contacts) == 0 || err != nil {
		debugContacts := []client.Contact{
			{Name: "status", Type: client.ContactPublicChat},
			{Name: "status-core", Type: client.ContactPublicChat},
			{Name: "testing-adamb", Type: client.ContactPublicChat},
			adambContact,
		}
		if err := db.SaveContacts(debugContacts); err != nil {
			exitErr(err)
		}
	}

	contacts := NewContactsViewController(&ViewController{vm, g, ViewContacts}, messenger)
	if err := contacts.Load(); err != nil {
		exitErr(err)
	}

	inputMultiplexer := NewInputMultiplexer()
	inputMultiplexer.AddHandler(DefaultMultiplexerPrefix, func(b []byte) error {
		log.Printf("default multiplexer handler")
		return chat.Send(b)
	})
	inputMultiplexer.AddHandler("/contact", ContactCmdFactory(contacts))
	inputMultiplexer.AddHandler("/request", RequestCmdFactory(chat))

	views := []*View{
		&View{
			Name:       ViewContacts,
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
						contact, ok := contacts.ContactByIdx(idx)
						if !ok {
							return errors.New("contact could not be found")
						}

						// We need to call Select asynchronously,
						// otherwise the main thread is blocked
						// and nothing is rendered.
						go func() {
							if err := chat.Select(contact); err != nil {
								log.Printf("[GetLineHandler] error selecting a chat: %v", err)
							}
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
						params := chat.RequestOptions(true)

						if err := notifications.Debug("Messages request", fmt.Sprintf("%v", params)); err != nil {
							return err
						}

						// RequestMessages needs to be called asynchronously,
						// otherwise the main thread is blocked
						// and nothing is rendered.
						go func() {
							if err := chat.RequestMessages(params); err != nil {
								_ = notifications.Error("Request failed", fmt.Sprintf("%v", err))
							}
						}()

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
			Name:      ViewInput,
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
		exitErr(err)
	}

	if err := vm.SetGlobalKeybindings(bindings); err != nil {
		exitErr(err)
	}

	// Put contacts into the view.
	contacts.Refresh()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		exitErr(err)
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
