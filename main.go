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

	"github.com/google/uuid"
	"github.com/jroimartin/gocui"
	"github.com/peterbourgon/ff"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/zaputil"
	"github.com/status-im/status-go/signal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var g *gocui.Gui

var (
	fs       = flag.NewFlagSet("status-term-client", flag.ExitOnError)
	logLevel = fs.String("log-level", "INFO", "log level")

	keyHex = fs.String("keyhex", "", "pass a private key in hex")
	noUI   = fs.Bool("no-ui", false, "disable UI")

	// flags acting like commands
	createKeyPair = fs.Bool("create-key-pair", false, "creates and prints a key pair instead of running")

	// flags for in-proc node
	dataDir        = fs.String("data-dir", filepath.Join(os.TempDir(), "status-term-client"), "data directory for Ethereum node")
	installationID = fs.String("installation-id", uuid.New().String(), "the installationID to be used")
	noNamespace    = fs.Bool("no-namespace", false, "disable data dir namespacing with public key")
	fleet          = fs.String("fleet", params.FleetStaging, fmt.Sprintf("Status nodes cluster to connect to: %s", []string{params.FleetBeta, params.FleetStaging}))
	configFile     = fs.String("node-config", "", "a JSON file with node config")
	listenAddr     = fs.String("listen-addr", ":30303", "The address the Ethereum node should be listening to")
	datasync       = fs.Bool("datasync", false, "enable datasync")

	// flags for external node
	providerURI = fs.String("provider", "", "an URI pointing at a provider")

	useNimbus = fs.Bool("nimbus", false, "use Nimbus node")
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

	if *useNimbus {
		if err := startNimbus(privateKey, *listenAddr, *fleet == params.FleetStaging); err != nil {
			exitErr(err)
		}
	}

	// Prefix data directory with a public key.
	// This is required because it's not possible
	// or advised to share data between different
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

	// Forward standard logger output.
	log.SetOutput(clientLogFile)

	// Create zap logger.
	if err := zaputil.RegisterJSONHexEncoder(); err != nil {
		exitErr(err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	cfg.OutputPaths = []string{clientLogFile.Name()}
	cfg.Encoding = "json-hex"
	logger, err := cfg.Build()
	if err != nil {
		exitErr(fmt.Errorf("failed to create logger: %v", err))
	}

	// Status node logs.
	nodeLogPath := filepath.Join(*dataDir, "status.log")
	err = logutils.OverrideRootLog(true, *logLevel, logutils.FileOptions{Filename: nodeLogPath}, false)
	if err != nil {
		exitErr(fmt.Errorf("failed to override root log: %v", err))
	}

	// initialize protocol
	var (
		messenger *protocol.Messenger
		pollFunc  func()
	)

	if *providerURI != "" {
		messenger, err = createMessengerWithURI(*providerURI)
		if err != nil {
			exitErr(err)
		}
	} else {
		messengerDBPath := filepath.Join(*dataDir, "messenger.sql")
		messenger, pollFunc, err = createMessengerInProc(privateKey, messengerDBPath, logger)
		if err != nil {
			exitErr(err)
		}
	}

	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)

	ossignal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Error("received signal", zap.String("signal", sig.String()))
		done <- true
	}()

	logger.Info("starting UI...")

	if *noUI {
		<-done
		return
	}

	go func() {
		<-done
		exitErr(errors.New("exit with signal"))
	}()

	if err := setupGUI(privateKey, messenger, logger); err != nil {
		exitErr(err)
	}

	if err := messenger.Init(); err != nil {
		exitErr(err)
	}

	cancelPolling := make(chan struct{}, 1)
	startPolling(g, pollFunc, 50*time.Millisecond, cancelPolling)
	defer close(cancelPolling)
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		close(cancelPolling)
		exitErr(err)
	}
	g.Close()
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

func createMessengerInProc(pk *ecdsa.PrivateKey, dbPath string, logger *zap.Logger) (*protocol.Messenger, func(), error) {
	// collect mail server request signals
	signalsForwarder := newSignalForwarder()
	go signalsForwarder.Start()

	// setup signals handler
	signal.SetDefaultNodeNotificationHandler(
		filterMailTypesHandler(signalsForwarder.in),
	)

	var (
		node types.Node
		err  error
	)
	if *useNimbus {
		node = newNimbusNodeWrapper()
	} else {
		if node, err = newGethNodeWrapper(pk); err != nil {
			exitErr(err)
		}
	}

	options := []protocol.Option{
		protocol.WithCustomLogger(logger),
		protocol.WithDatabaseConfig(dbPath, ""),
		protocol.WithMessagesPersistenceEnabled(),
	}

	if *datasync {
		options = append(options, protocol.WithDatasync())
	}

	messenger, err := protocol.NewMessenger(
		pk,
		node,
		*installationID,
		options...,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create Messenger")
	}

	if err := messenger.Init(); err != nil {
		return nil, nil, err
	}

	// protocolGethService.SetMessenger(messenger)

	var whisper types.Whisper
	if whisper, err = node.GetWhisper(nil); err != nil {
		exitErr(err)
	}

	return messenger, whisper.Poll, nil
}

func setupGUI(privateKey *ecdsa.PrivateKey, messenger *protocol.Messenger, logger *zap.Logger) error {
	var err error

	// global
	g, err = gocui.NewGui(gocui.Output256)
	if err != nil {
		return err
	}

	// prepare views
	vm := NewViewManager(nil, g, logger)

	notifications := NewNotificationViewController(&ViewController{vm, g, ViewNotification})

	chatsVC := NewChatsViewController(&ViewController{vm, g, ViewChats}, messenger, logger)
	if err := chatsVC.LoadAndRefresh(); err != nil {
		return errors.Wrap(err, "failed to load chats")
	}

	messagesVC := NewMessagesViewController(
		&ViewController{vm, g, ViewChat},
		privateKey,
		messenger,
		logger,
		func() {
			if err := chatsVC.LoadAndRefresh(); err != nil {
				logger.Error("failed to load and refresh chats", zap.Error(err))
			}
		},
		func(err error) {
			_ = notifications.Error("Chat error", fmt.Sprintf("%v", err))
		},
	)
	err = messagesVC.Start()
	if err != nil {
		return err
	}

	inputMultiplexer := NewInputMultiplexer()
	inputMultiplexer.AddHandler(DefaultMultiplexerPrefix, func(b []byte) error {
		logger.Info("default multiplexer handler")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		response, err := messagesVC.Send(ctx, string(b))
		logger.Info("SENT MESSAGE", zap.Any("RESPOSNE", response))
		return err
	})
	inputMultiplexer.AddHandler("/chat", ChatCmdFactory(chatsVC, messagesVC))
	// inputMultiplexer.AddHandler("/request", RequestCmdFactory(chatVC))

	views := []*View{
		{
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
				{
					Key:     gocui.KeyArrowDown,
					Mod:     gocui.ModNone,
					Handler: CursorDownHandler,
				},
				{
					Key:     gocui.KeyArrowUp,
					Mod:     gocui.ModNone,
					Handler: CursorUpHandler,
				},
				{
					Key: gocui.KeyEnter,
					Mod: gocui.ModNone,
					Handler: GetLineHandler(func(idx int, _ string) error {
						selectedChat, ok := chatsVC.ChatByIdx(idx)
						if !ok {
							return errors.New("chat could not be found")
						}

						// We need to call Select asynchronously,
						// otherwise the main thread is blocked
						// and nothing is rendered.
						go func() {
							messagesVC.Select(selectedChat)
						}()

						return nil
					}),
				},
			},
		},
		{
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
				{
					Key:     gocui.KeyArrowDown,
					Mod:     gocui.ModNone,
					Handler: CursorDownHandler,
				},
				{
					Key:     gocui.KeyArrowUp,
					Mod:     gocui.ModNone,
					Handler: CursorUpHandler,
				},
				{
					Key: gocui.KeyHome,
					Mod: gocui.ModNone,
					Handler: func(g *gocui.Gui, v *gocui.View) error {
						return HomeHandler(g, v)
					},
				},
				{
					Key:     gocui.KeyEnd,
					Mod:     gocui.ModNone,
					Handler: EndHandler,
				},
			},
		},
		{
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
				{
					Key:     gocui.KeyEnter,
					Mod:     gocui.ModNone,
					Handler: inputMultiplexer.BindingHandler,
				},
				{
					Key:     gocui.KeyEnter,
					Mod:     gocui.ModAlt,
					Handler: MoveToNewLineHandler,
				},
			},
		},
		{
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
				{
					Key: gocui.KeyEnter,
					Mod: gocui.ModNone,
					Handler: func(g *gocui.Gui, v *gocui.View) error {
						logger.Info("Notification Enter binding")

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
		{
			Key:     gocui.KeyCtrlC,
			Mod:     gocui.ModNone,
			Handler: QuitHandler,
		},
		{
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
