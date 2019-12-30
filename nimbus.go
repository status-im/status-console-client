// +build nimbus

package main

import (
	"crypto/ecdsa"

	nimbusbridge "github.com/status-im/status-go/eth-node/bridge/nimbus"
	"github.com/status-im/status-go/eth-node/types"
)

func newNimbusNodeWrapper() (types.Node, func()) {
	nimbusNode := nimbusbridge.NewNodeBridge()
	return nimbusNode, nimbusNode.Stop
}

func startNimbus(node types.Node, privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	return node.(nimbusbridge.Node).StartNimbus(privateKey, listenAddr, staging)
}
