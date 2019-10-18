// +build geth !nimbus

package main

import (
	"crypto/ecdsa"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/internal/gethservice"
	"github.com/status-im/status-go/node"
	status "github.com/status-im/status-protocol-go"
	gethbridge "github.com/status-im/status-protocol-go/bridge/geth"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
)

func newGethWhisperWrapper(pk *ecdsa.PrivateKey) (whispertypes.Whisper, error) {
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

	shhService, err := statusNode.WhisperService()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Whisper service")
	}

	return gethbridge.NewGethWhisperWrapper(shhService), nil
}

func createMessengerWithURI(uri string) (*status.Messenger, error) {
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
