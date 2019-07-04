package main

import (
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

	"github.com/ethereum/go-ethereum/crypto"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/messaging/chat"
	"github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/messaging/multidevice"
	"github.com/status-im/status-go/messaging/publisher"
	"github.com/status-im/status-go/messaging/sharedsecret"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"

	"github.com/jroimartin/gocui"
	"github.com/peterbourgon/ff"
	"github.com/pkg/errors"

	datasyncnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/state"
	"github.com/status-im/mvds/store"

	"github.com/status-im/status-console-client/protocol/adapter"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/datasync"
	dspeer "github.com/status-im/status-console-client/protocol/datasync/peer"
	"github.com/status-im/status-console-client/protocol/gethservice"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"
)

var g *gocui.Gui

var (
	fs       = flag.NewFlagSet("status-term-client", flag.ExitOnError)
	logLevel = fs.String("log-level", "INFO", "log level")

	keyHex = fs.String("keyhex", "", "pass a private key in hex")
	noUI   = fs.Bool("no-ui", false, "disable UI")

	// flags acting like commands
	createKeyPair = fs.Bool("create-key-pair", false, "creates and prints a key pair instead of running")
	addContact    = fs.String("add-contact", "", "add contact using format: type,name[,public-key] where type can be 'private' or 'public' and 'public-key' is required for 'private' type")

	// flags for in-proc node
	dataDir         = fs.String("data-dir", filepath.Join(os.TempDir(), "status-term-client"), "data directory for Ethereum node")
	noNamespace     = fs.Bool("no-namespace", false, "disable data dir namespacing with public key")
	fleet           = fs.String("fleet", params.FleetBeta, fmt.Sprintf("Status nodes cluster to connect to: %s", []string{params.FleetBeta, params.FleetStaging}))
	configFile      = fs.String("node-config", "", "a JSON file with node config")
	pfsEnabled      = fs.Bool("pfs", false, "enable PFS")
	dataSyncEnabled = fs.Bool("ds", false, "enable data sync")

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
	dbKey := crypto.PubkeyToAddress(privateKey.PublicKey).String()
	dbPath := filepath.Join(*dataDir, "db.sql")
	db, err := client.InitializeDB(dbPath, dbKey)
	if err != nil {
		exitErr(err)
	}
	defer db.Close()

	// Log the current contact info in two places for easy retrieval.
	fmt.Printf("Contact address: %#x\n", crypto.FromECDSAPub(&privateKey.PublicKey))
	log.Printf("contact address: %#x", crypto.FromECDSAPub(&privateKey.PublicKey))

	// Manage initial contacts.
	if contacts, err := db.Contacts(); len(contacts) == 0 || err != nil {
		debugContacts := []client.Contact{
			{Name: "status", Type: client.ContactPublicRoom, Topic: "status"},
			{Name: "status-core", Type: client.ContactPublicRoom, Topic: "status-core"},
		}
		uniqueContacts := []client.Contact{}
		for _, c := range debugContacts {
			exist, err := db.ContactExist(c)
			if err != nil {
				exitErr(err)
			}
			if !exist {
				uniqueContacts = append(uniqueContacts, c)
			}
		}
		if len(uniqueContacts) != 0 {
			if err := db.SaveContacts(uniqueContacts); err != nil {
				exitErr(err)
			}
		}
	}

	// Handle add contact.
	if *addContact != "" {
		options := strings.Split(*addContact, ",")

		var c client.Contact

		if len(options) == 2 && options[0] == "public" {
			c = client.Contact{
				Name:  options[1],
				Type:  client.ContactPublicRoom,
				Topic: options[1],
			}
		} else if len(options) == 3 && options[0] == "private" {
			c, err = client.CreateContactPrivate(options[1], options[2], client.ContactAdded)
			if err != nil {
				exitErr(err)
			}
		} else {
			exitErr(errors.Errorf("invalid -add-contact value"))
		}

		exists, err := db.ContactExist(c)
		if err != nil {
			exitErr(err)
		}
		if !exists {
			if err := db.SaveContacts([]client.Contact{c}); err != nil {
				exitErr(err)
			}
		}
	}

	// initialize protocol
	var (
		messenger *client.Messenger
	)

	if *providerURI != "" {
		messenger, err = createMessengerWithURI(*providerURI, privateKey, db)
		if err != nil {
			exitErr(err)
		}
	} else {
		messenger, err = createMessengerInProc(privateKey, db)
		if err != nil {
			exitErr(err)
		}
	}

	// run in a goroutine to show the UI faster
	go func() {
		if err := messenger.Start(); err != nil {
			exitErr(err)
		}
	}()

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

		if err := setupGUI(privateKey, messenger); err != nil {
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

func createMessengerWithURI(uri string, pk *ecdsa.PrivateKey, db client.Database) (*client.Messenger, error) {
	_, err := rpc.Dial(*providerURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial")
	}

	// TODO: provide Mail Servers in a different way.
	_, err = generateStatusNodeConfig(*dataDir, *fleet, *configFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate node config")
	}

	// TODO

	return nil, errors.New("not implemented")
}

func createMessengerInProc(pk *ecdsa.PrivateKey, db client.Database) (*client.Messenger, error) {
	// collect mail server request signals
	signalsForwarder := newSignalForwarder()
	go signalsForwarder.Start()

	// setup signals handler
	signal.SetDefaultNodeNotificationHandler(
		filterMailTypesHandler(signalsForwarder.in),
	)

	nodeConfig, err := generateStatusNodeConfig(*dataDir, *fleet, *configFile)
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

	shhExtService, err := statusNode.ShhExtService()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get shhext service")
	}

	transp, err := transport.NewWhisperServiceTransport(
		&server{node: statusNode},
		nodeConfig.ClusterConfig.TrustedMailServers,
		shhService,
		shhExtService,
		pk,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create WhisperService transport")
	}

	var (
		protocolAdapter protocol.Protocol
	)

	if *dataSyncEnabled {
		dataSyncTransport := datasync.NewDataSyncNodeTransport(transp)
		dataSyncStore := store.NewDummyStore()
		dataSyncNode := datasyncnode.NewNode(
			&dataSyncStore,
			dataSyncTransport,
			state.NewSyncState(), // @todo sqlite syncstate
			datasync.CalculateSendTime,
			0,
			dspeer.PublicKeyToPeerID(pk.PublicKey),
			datasyncnode.BATCH,
		)

		dataSyncNode.Start()

		protocolAdapter = adapter.NewDataSyncWhisperAdapter(dataSyncNode, transp, dataSyncTransport)
	} else {
		publisher := publisher.New(
			shhService,
			publisher.Config{PFSEnabled: *pfsEnabled},
		)

		databasesDir := filepath.Join(*dataDir, "databases")
		if err := os.MkdirAll(databasesDir, 0755); err != nil {
			exitErr(errors.Wrap(err, "failed to create databases dir"))
		}
		persistence, err := initPersistence(databasesDir)

		var protocol *chat.ProtocolService

		if *pfsEnabled {
			protocol, err = initProtocol(persistence)
			if err != nil {
				exitErr(errors.Wrap(err, "initialize protocol"))
			}

			log.Printf("Protocol has been initialized")
		}

		// Initialize sharedsecret
		sharedSecretService := sharedsecret.NewService(persistence.GetSharedSecretStorage())

		// Initialize filter
		filterService := filter.New(shhService, filter.NewSQLLitePersistence(persistence.DB), sharedSecretService)
		if _, err := filterService.Init(nil); err != nil {
			return nil, errors.Wrap(err, "failed to init Filter service")
		}

		// Init publisher
		publisher.Init(persistence.DB, protocol, filterService)
		if err := publisher.Start(func() bool { return true }, true); err != nil {
			return nil, errors.Wrap(err, "failed to start Publisher")
		}

		protocolAdapter = adapter.NewProtocolWhisperAdapter(transp, publisher)
	}

	messenger := client.NewMessenger(pk, protocolAdapter, db)

	protocolGethService.SetProtocol(protocolAdapter)
	protocolGethService.SetMessenger(messenger)

	return messenger, nil
}

func setupGUI(privateKey *ecdsa.PrivateKey, messenger *client.Messenger) error {
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

	contacts := NewContactsViewController(&ViewController{vm, g, ViewContacts}, messenger)
	if err := contacts.LoadAndRefresh(); err != nil {
		return err
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
						params, err := chat.RequestOptions(false)
						if err != nil {
							return err
						}

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

func initPersistence(baseDir string) (*chat.SQLLitePersistence, error) {
	const (
		// TODO: manage these values properly
		dbFileName   = "pfs_v1.db"
		sqlSecretKey = "enc-key-abc"
	)

	dbPath := filepath.Join(baseDir, dbFileName)
	return chat.NewSQLLitePersistence(dbPath, sqlSecretKey)
}

func initProtocol(p *chat.SQLLitePersistence) (*chat.ProtocolService, error) {
	const (
		installationID   = "installation-1"
		maxInstallations = 3
	)

	addedBundlesHandler := func(addedBundles []*multidevice.Installation) {
		log.Printf("added bundles: %v", addedBundles)
	}

	sharedSecretHandler := func(sharedSecrets []*sharedsecret.Secret) {
		log.Printf("new shared secrets: %v", sharedSecrets)
	}

	sharedSecretService := sharedsecret.NewService(p.GetSharedSecretStorage())

	multideviceConfig := &multidevice.Config{
		InstallationID:   installationID,
		ProtocolVersion:  chat.ProtocolVersion,
		MaxInstallations: maxInstallations,
	}
	multideviceService := multidevice.New(
		multideviceConfig,
		p.GetMultideviceStorage(),
	)

	return chat.NewProtocolService(
		chat.NewEncryptionService(
			p,
			chat.DefaultEncryptionServiceConfig(installationID),
		),
		sharedSecretService,
		multideviceService,
		addedBundlesHandler,
		sharedSecretHandler,
	), nil
}
