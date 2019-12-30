// +build !nimbus

package main

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/types"
)

const noNimbusError = "executable needs to be built with -tags nimbus"

func newNimbusNodeWrapper() (types.Node, func()) {
	panic(noNimbusError)
}

func startNimbus(node types.Node, privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	panic(noNimbusError)
}
