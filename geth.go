// +build geth !nimbus

package main

import (
	"crypto/ecdsa"

	"github.com/pkg/errors"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/protocol"

	"github.com/status-im/status-console-client/internal/gethservice"
)

func newGethNodeWrapper(pk *ecdsa.PrivateKey) (types.Node, error) {
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

	if err := statusNode.Start(nodeConfig, nil, services...); err != nil {
		return nil, errors.Wrap(err, "failed to start node")
	}

	return gethbridge.NewNodeBridge(statusNode.GethNode()), nil
}

func createMessengerWithURI(uri string) (*protocol.Messenger, error) {
	_, err := rpc.Dial(uri)
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
