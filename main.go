package main

import (
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jroimartin/gocui"
	"github.com/peterbourgon/ff"
	"github.com/status-im/status-go/params"
)

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	fs := flag.NewFlagSet("status-term-client", flag.ExitOnError)

	var (
		// flags acting like commands
		createKeyPair = fs.Bool("create-key-pair", false, "creates and prints a key pair instead of running")

		// runtime flags
		dataDir    = fs.String("data-dir", filepath.Join(os.TempDir(), "status-term-client"), "data directory for Ethereum node")
		fleet      = fs.String("fleet", params.FleetBeta, fmt.Sprintf("Status nodes cluster to connect to: %s", []string{params.FleetBeta, params.FleetStaging}))
		configFile = fs.String("node-config", "", "a JSON file with node config")
		keyHex     = fs.String("keyhex", "", "pass a private key in hex")
	)

	ff.Parse(fs, os.Args[1:])

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
		exitErr(errors.New("private key is required"))
	}

	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		exitErr(err)
	}
	defer g.Close()

	// start Status node
	node := NewNode()
	if err := node.Start(*dataDir, *fleet, *configFile); err != nil {
		exitErr(err)
	}

	// prepare views
	vm := NewViewManager(nil, g)

	chat, err := NewChatViewController(&ViewController{vm, g, ViewChat}, privateKey, node)
	if err != nil {
		exitErr(err)
	}

	contacts := NewContactsViewController(
		&ViewController{vm, g, ViewContacts},
		[]Contact{
			{"status", ContactPublicChat},
			{"status-core", ContactPublicChat},
			{"testing-adamb", ContactPublicChat},
		},
	)

	views := []*View{
		&View{
			Name:       ViewContacts,
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
						return chat.Select(contact)
					}),
				},
			},
		},
		&View{
			Name:       ViewChat,
			Cursor:     true,
			Autoscroll: true,
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
			},
			OnActivate: func(self *View) {
				self.Autoscroll = false
			},
			OnDeactivate: func(self *View) {
				self.Autoscroll = true
			},
		},
		&View{
			Name:      ViewInput,
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
					Handler: EnterInputHandler(chat),
				},
				Binding{
					Key:     gocui.KeyEnter,
					Mod:     gocui.ModAlt,
					Handler: MoveToNewLineHandler,
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
	fmt.Println(err)
	os.Exit(1)
}
