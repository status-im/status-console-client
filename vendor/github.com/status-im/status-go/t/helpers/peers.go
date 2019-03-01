package helpers

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	// ErrNoRunningNode node is not running.
	ErrNoRunningNode = errors.New("there is no running node")
	// ErrEmptyPeerURL provided peer URL is empty
	ErrEmptyPeerURL = errors.New("empty peer url")
)

// waitForPeer waits for a peer to be added
func waitForPeer(server *p2p.Server, peerURL string, typ p2p.PeerEventType, t time.Duration, subscribed chan struct{}) error {
	if server == nil {
		return ErrNoRunningNode
	}
	if peerURL == "" {
		return ErrEmptyPeerURL
	}
	parsedPeer, err := enode.ParseV4(peerURL)
	if err != nil {
		return err
	}

	ch := make(chan *p2p.PeerEvent)
	subscription := server.SubscribeEvents(ch)
	defer subscription.Unsubscribe()
	close(subscribed)

	for {
		select {
		case ev := <-ch:
			if ev.Type == typ && ev.Peer == parsedPeer.ID() {
				return nil
			}
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
		case <-time.After(t):
			return fmt.Errorf("wait for peer %s: timeout", u)
		}
	}
}

// WaitForPeerAsync waits for a peer to be added asynchronously
func WaitForPeerAsync(p *p2p.Server, u string, e p2p.PeerEventType, t time.Duration) <-chan error {
	subscribed := make(chan struct{})
	errCh := make(chan error)
	go func() {
		errCh <- waitForPeer(p, u, e, t, subscribed)
	}()
	<-subscribed
	return errCh
}
