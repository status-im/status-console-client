package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/t/helpers"

	whisper "github.com/status-im/whisper/whisperv6"
)

// WhisperServiceAdapter is an adapter for Whisper service
// the implements Chat interface. It requires a Whisper service .
type WhisperServiceAdapter struct {
	node *node.StatusNode
	shh  *whisper.Whisper
}

// WhisperServiceAdapter must implement Chat interface.
var _ Chat = (*WhisperServiceAdapter)(nil)

// NewWhisperServiceAdapter returns a new WhisperServiceAdapter.
func NewWhisperServiceAdapter(node *node.StatusNode, shh *whisper.Whisper) *WhisperServiceAdapter {
	return &WhisperServiceAdapter{
		node: node,
		shh:  shh,
	}
}

// SubscribePublicChat subscribes to a public chat using the Whisper service.
func (a *WhisperServiceAdapter) SubscribePublicChat(ctx context.Context, name string, in chan<- *ReceivedMessage) (*Subscription, error) {
	// TODO: add cache
	symKeyID, err := a.shh.AddSymKeyFromPassword(name)
	if err != nil {
		return nil, err
	}
	symKey, err := a.shh.GetSymKey(symKeyID)
	if err != nil {
		return nil, err
	}

	// TODO: add cache
	topic, err := PublicChatTopic(name)
	if err != nil {
		return nil, err
	}

	filterID, err := a.shh.Subscribe(&whisper.Filter{
		KeySym:   symKey,
		Topics:   [][]byte{topic[:]},
		PoW:      0,
		AllowP2P: true,
	})
	if err != nil {
		return nil, err
	}

	subMessages := newWhisperSubscription(a.shh, filterID)
	sub := NewSubscription()

	go func() {
		defer subMessages.Unsubscribe()

		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				messages, err := subMessages.Messages()
				if err != nil {
					sub.cancel(err)
					return
				}

				sort.Slice(messages, func(i, j int) bool {
					return messages[i].Decoded.Clock < messages[j].Decoded.Clock
				})

				for _, m := range messages {
					in <- m
				}
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

// SendPublicMessage sends a new message using the Whisper service.
func (a *WhisperServiceAdapter) SendPublicMessage(
	ctx context.Context, name string, data []byte, identity *ecdsa.PrivateKey,
) (string, error) {
	// TODO: add cache
	keyID, err := a.shh.AddKeyPair(identity)
	if err != nil {
		return "", err
	}

	// TODO: add cache
	symKeyID, err := a.shh.AddSymKeyFromPassword(name)
	if err != nil {
		return "", err
	}

	// TODO: add cache
	topic, err := PublicChatTopic(name)
	if err != nil {
		return "", err
	}

	// Only public Whisper API implements logic to send messages.
	shhAPI := whisper.NewPublicWhisperAPI(a.shh)
	hash, err := shhAPI.Post(ctx, whisper.NewMessage{
		SymKeyID:  symKeyID,
		TTL:       60,
		Topic:     topic,
		Payload:   data,
		PowTarget: 2.0,
		PowTime:   5,
		Sig:       keyID,
	})

	return hash.String(), err
}

// RequestPublicMessages requests messages from mail servers.
func (a *WhisperServiceAdapter) RequestPublicMessages(
	ctx context.Context, name string, params RequestMessagesParams,
) error {
	// TODO: add cache
	topic, err := PublicChatTopic(name)
	if err != nil {
		return err
	}

	shhextService, err := a.node.ShhExtService()
	if err != nil {
		return err
	}
	shhextAPI := shhext.NewPublicAPI(shhextService)

	// TODO: remove from here. MailServerEnode must be provided in the params.
	config := a.node.Config()
	mailServerEnode := randomItem(config.ClusterConfig.TrustedMailServers)
	errCh := helpers.WaitForPeerAsync(
		a.node.GethNode().Server(),
		mailServerEnode,
		p2p.PeerEventTypeAdd,
		time.Second*5,
	)
	if err := a.node.AddPeer(mailServerEnode); err != nil {
		return err
	}
	if err := <-errCh; err != nil {
		return err
	}

	mailServerSymKeyID, err := a.shh.AddSymKeyFromPassword(MailServerPassword)
	if err != nil {
		return err
	}

	_, err = shhextAPI.RequestMessages(ctx, shhext.MessagesRequest{
		MailServerPeer: mailServerEnode,
		From:           uint32(params.From),  // TODO: change to int in status-go
		To:             uint32(params.To),    // TODO: change to int in status-go
		Limit:          uint32(params.Limit), // TODO: change to int in status-go
		Topics:         []whisper.TopicType{topic},
		SymKeyID:       mailServerSymKeyID,
	})

	// TODO: wait for the request to finish before returning
	return err
}

// whisperSubscription encapsulates a Whisper filter.
type whisperSubscription struct {
	shh      *whisper.Whisper
	filterID string
}

// newWhisperSubscription returns a new whisperSubscription.
func newWhisperSubscription(shh *whisper.Whisper, filterID string) *whisperSubscription {
	return &whisperSubscription{shh, filterID}
}

// Messages retrieves a list of messages for a given filter.
func (s whisperSubscription) Messages() ([]*ReceivedMessage, error) {
	f := s.shh.GetFilter(s.filterID)
	if f == nil {
		return nil, errors.New("filter does not exist")
	}

	items := f.Retrieve()
	result := make([]*ReceivedMessage, len(items))

	for i, item := range items {
		decoded, err := DecodeMessage(item.Payload)
		if err != nil {
			log.Printf("failed to decode message: %v", err)
			continue
		}

		result[i] = &ReceivedMessage{
			Decoded: decoded,
			Src:     item.Signature,
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Decoded.Clock < result[j].Decoded.Clock
	})

	return result, nil
}

// Unsubscribe removes the subscription.
func (s whisperSubscription) Unsubscribe() error {
	return s.shh.Unsubscribe(s.filterID)
}

func randomItem(items []string) string {
	l := len(items)
	return items[rand.Intn(l)]
}
