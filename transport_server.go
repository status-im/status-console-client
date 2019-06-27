package main

import (
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/node"
)

type server struct {
	node *node.StatusNode
}

func (s *server) AddPeer(peer string) error {
	return s.node.AddPeer(peer)
}

func (s *server) Connected(id enode.ID) (bool, error) {
	geth := s.node.GethNode()
	if geth == nil {
		return false, errors.New("devp2p server isn't running")
	}
	for _, p := range geth.Server().Peers() {
		if p.ID() == id {
			return true, nil
		}
	}
	return false, nil
}
