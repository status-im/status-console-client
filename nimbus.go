// +build nimbus

package main

import (
	"time"

	"crypto/ecdsa"

	"github.com/jroimartin/gocui"
	nimbusbridge "github.com/status-im/status-go/eth-node/bridge/nimbus"
	"github.com/status-im/status-go/eth-node/types"
)

func init() {
	nimbusbridge.Init()
}

func startNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	return nimbusbridge.StartNimbus(privateKey, listenAddr, staging)
}

func startPolling(g *gocui.Gui, pollFunc func(), delay time.Duration, cancel <-chan struct{}) {
	if pollFunc != nil {
		// Start a worker goroutine to periodically schedule polling on the UI thread
		go func() {
			for {
				schedulePoll(g, pollFunc)
				select {
				case <-time.After(delay):
				case <-cancel:
					return
				}
			}
		}()
	}
}

func schedulePoll(g *gocui.Gui, pollFunc func()) {
	if pollFunc == nil {
		return
	}

	g.Update(func(g *gocui.Gui) error {
		pollFunc()
		return nil
	})
}

func newNimbusNodeWrapper() types.Node {
	return nimbusbridge.NewNodeBridge()
}
