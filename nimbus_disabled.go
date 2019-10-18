// +build !nimbus

package main

import (
	"crypto/ecdsa"
	"time"

	"github.com/jroimartin/gocui"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
)

const noNimbusError = "executable needs to be built with -tags nimbus"

func startNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	panic(noNimbusError)
}

func startPolling(g *gocui.Gui, pollFunc func(), delay time.Duration, cancel <-chan struct{}) {}

func newNimbusWhisperWrapper() whispertypes.Whisper {
	panic(noNimbusError)
}
