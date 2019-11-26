// +build !nimbus

package main

import (
	"crypto/ecdsa"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/status-im/status-go/eth-node/types"
)

const noNimbusError = "executable needs to be built with -tags nimbus"

func startNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	panic(noNimbusError)
}

func startPolling(g *gocui.Gui, pollFunc func(), delay time.Duration, cancel <-chan struct{}) {}

func newNimbusNodeWrapper() types.Node {
	panic(noNimbusError)
}
