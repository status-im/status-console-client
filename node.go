package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	stdlog "log"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/status-im/status-term-client/protocol/v1"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/t/helpers"
	whisper "github.com/status-im/whisper/whisperv6"
)

func init() {
	if err := logutils.OverrideRootLog(true, "DEBUG", "", false); err != nil {
		stdlog.Fatalf("failed to override root log: %v\n", err)
	}
}

// WhisperSubscription encapsulates a Whisper filter.
type WhisperSubscription struct {
	shh      *whisper.Whisper
	filterID string
}

// NewWhisperSubscription returns a new WhisperSubscription.
func NewWhisperSubscription(shh *whisper.Whisper, filterID string) *WhisperSubscription {
	return &WhisperSubscription{shh, filterID}
}

// Messages retrieves a list of messages for a given filter.
func (s WhisperSubscription) Messages() ([]*ReceivedMessage, error) {
	f := s.shh.GetFilter(s.filterID)
	if f == nil {
		return nil, errors.New("filter does not exist")
	}

	items := f.Retrieve()
	result := make([]*ReceivedMessage, len(items))

	for i, item := range items {
		decoded, err := protocol.DecodeMessage(item.Payload)
		if err != nil {
			log.Printf("failed to decode message: %s", item.Payload)
			continue
		}

		result[i] = &ReceivedMessage{
			Received: item,
			Decoded:  decoded,
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Decoded.Clock < result[j].Decoded.Clock
	})

	return result, nil
}

// Unsubscribe removes the subscription.
func (s WhisperSubscription) Unsubscribe() error {
	return s.shh.Unsubscribe(s.filterID)
}

// Node is an adapter between Chat interface and Status node.
type Node struct {
	node *node.StatusNode
}

// NewNode returns a new node.
func NewNode() *Node {
	return &Node{node: node.New()}
}

// Start starts the node.
// TODO: params should be taken from the config argument.
func (n *Node) Start(dataDir, fleet, configFile string) error {
	if err := os.MkdirAll(dataDir, os.ModeDir|0755); err != nil {
		return fmt.Errorf("failed to create a data dir: %v", err)
	}

	var configFiles []string
	if configFile != "" {
		configFiles = append(configFiles, configFile)
	}

	config, err := params.NewNodeConfigWithDefaultsAndFiles(
		dataDir,
		params.MainNetworkID,
		[]params.Option{params.WithFleet(fleet)},
		configFiles,
	)
	if err != nil {
		return fmt.Errorf("failed to create a config: %v", err)
	}
	return n.node.Start(config)
}

// AddKeyPair adds a key pair to the Whisper service.
func (n *Node) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	shh, err := n.node.WhisperService()
	if err != nil {
		return "", err
	}
	return shh.AddKeyPair(key)
}

// SubscribePublicChat subscribes to a public chat using the Whisper service.
func (n *Node) SubscribePublicChat(name string) (sub MessagesSubscription, err error) {
	shh, err := n.node.WhisperService()
	if err != nil {
		return
	}

	// TODO: add cache
	symKeyID, err := shh.AddSymKeyFromPassword(name)
	if err != nil {
		return
	}
	symKey, err := shh.GetSymKey(symKeyID)
	if err != nil {
		return
	}

	// TODO: add cache
	topic, err := protocol.PublicChatTopic(name)
	if err != nil {
		return
	}

	filterID, err := shh.Subscribe(&whisper.Filter{
		KeySym:   symKey,
		Topics:   [][]byte{topic[:]},
		PoW:      0,
		AllowP2P: true,
	})
	if err != nil {
		return
	}

	return NewWhisperSubscription(shh, filterID), nil
}

// SendPublicMessage sends a new message using the Whisper service.
func (n *Node) SendPublicMessage(name string, data []byte, identity Identity) (string, error) {
	whisperService, err := n.node.WhisperService()
	if err != nil {
		return "", err
	}

	// TODO: add cache
	keyID, err := whisperService.AddKeyPair(identity)
	if err != nil {
		return "", err
	}

	// TODO: add cache
	symKeyID, err := whisperService.AddSymKeyFromPassword(name)
	if err != nil {
		return "", err
	}

	// TODO: add cache
	topic, err := protocol.PublicChatTopic(name)
	if err != nil {
		return "", err
	}

	shh := whisper.NewPublicWhisperAPI(whisperService)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	hash, err := shh.Post(ctx, whisper.NewMessage{
		SymKeyID:  symKeyID,
		TTL:       60,
		Topic:     topic,
		Payload:   data,
		PowTarget: 2.0,
		PowTime:   5,
		Sig:       keyID,
	})
	if err == nil {
		stdlog.Printf("sent a message with hash %s", hash.String())
	}

	return hash.String(), err
}

// RequestPublicMessages requests messages from mail servers.
func (n *Node) RequestPublicMessages(chatName string, params RequestMessagesParams) error {
	// TODO: add cache
	topic, err := protocol.PublicChatTopic(chatName)
	if err != nil {
		return err
	}

	shhService, err := n.node.WhisperService()
	if err != nil {
		return err
	}

	shhextService, err := n.node.ShhExtService()
	if err != nil {
		return err
	}
	shhextAPI := shhext.NewPublicAPI(shhextService)

	config := n.node.Config()
	mailServerEnode := randomItem(config.ClusterConfig.TrustedMailServers)
	errCh := helpers.WaitForPeerAsync(
		n.node.GethNode().Server(),
		mailServerEnode,
		p2p.PeerEventTypeAdd,
		time.Second*5,
	)
	if err := n.node.AddPeer(mailServerEnode); err != nil {
		return err
	}
	if err := <-errCh; err != nil {
		return err
	}

	mailServerSymKeyID, err := shhService.AddSymKeyFromPassword(protocol.MailServerPassword)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	hash, err := shhextAPI.RequestMessages(ctx, shhext.MessagesRequest{
		MailServerPeer: mailServerEnode,
		From:           uint32(params.From),  // TODO: change to int in status-go
		To:             uint32(params.To),    // TODO: change to int in status-go
		Limit:          uint32(params.Limit), // TODO: change to int in status-go
		Topics:         []whisper.TopicType{topic},
		SymKeyID:       mailServerSymKeyID,
	})
	if err == nil {
		stdlog.Printf("send a request for messages: %s", hash.String())
	}

	// TODO: wait for the request to finish before returning
	return err
}

func randomItem(items []string) string {
	l := len(items)
	return items[rand.Intn(l)]
}
