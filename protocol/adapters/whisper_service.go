package adapters

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/services/shhext"

	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/ethereum/go-ethereum/p2p"

	whisper "github.com/status-im/whisper/whisperv6"
)

type whisperServiceKeysManager struct {
	shh *whisper.Whisper

	passToSymMutex    sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *whisperServiceKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.shh.AddKeyPair(priv)
}

func (m *whisperServiceKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymMutex.Lock()
	defer m.passToSymMutex.Unlock()

	if val, ok := m.passToSymKeyCache[password]; ok {
		return val, nil
	}

	id, err := m.shh.AddSymKeyFromPassword(password)
	if err != nil {
		return id, err
	}

	m.passToSymKeyCache[password] = id

	return id, nil
}

func (m *whisperServiceKeysManager) GetRawSymKey(id string) ([]byte, error) {
	return m.shh.GetSymKey(id)
}

// WhisperServiceAdapter is an adapter for Whisper service
// the implements Protocol interface.
type WhisperServiceAdapter struct {
	node        *node.StatusNode
	shh         *whisper.Whisper
	keysManager *whisperServiceKeysManager

	selectedMailServerEnode string
}

// WhisperServiceAdapter must implement Protocol interface.
var _ protocol.Protocol = (*WhisperServiceAdapter)(nil)

// NewWhisperServiceAdapter returns a new WhisperServiceAdapter.
func NewWhisperServiceAdapter(node *node.StatusNode, shh *whisper.Whisper) *WhisperServiceAdapter {
	return &WhisperServiceAdapter{
		node:        node,
		shh:         shh,
		keysManager: &whisperServiceKeysManager{shh: shh},
	}
}

// Subscribe subscribes to a public chat using the Whisper service.
func (a *WhisperServiceAdapter) Subscribe(
	ctx context.Context,
	in chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter, err := createRichFilter(a.keysManager, options)
	if err != nil {
		return nil, err
	}

	filterID, err := a.shh.Subscribe(filter)
	if err != nil {
		return nil, err
	}

	subWhisper := newWhisperSubscription(a.shh, filterID)
	sub := protocol.NewSubscription()

	go func() {
		defer subWhisper.Unsubscribe() // nolint: errcheck

		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				messages, err := subWhisper.Messages()
				if err != nil {
					sub.Cancel(err)
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

// Send sends a new message using the Whisper service.
func (a *WhisperServiceAdapter) Send(
	ctx context.Context,
	data []byte,
	options protocol.SendOptions,
) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	messag, err := createRichWhisperNewMessage(a.keysManager, data, options)
	if err != nil {
		return nil, err
	}

	// Only public Whisper API implements logic to send messages.
	shhAPI := whisper.NewPublicWhisperAPI(a.shh)
	return shhAPI.Post(ctx, messag)
}

// Request requests messages from mail servers.
func (a *WhisperServiceAdapter) Request(ctx context.Context, options protocol.RequestOptions) error {
	if err := options.Validate(); err != nil {
		return err
	}

	// TODO: remove from here. MailServerEnode must be provided in the params.
	enode, err := a.selectAndAddMailServer()
	if err != nil {
		return err
	}

	keyID, err := a.keysManager.AddOrGetSymKeyFromPassword(MailServerPassword)
	if err != nil {
		return err
	}

	req, err := createShhextRequestMessagesParam(enode, keyID, options)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = a.requestMessages(ctx, req, true)

	log.Printf("[WhisperServiceAdapter::Request] took %s", time.Since(now))

	return err
}

func (a *WhisperServiceAdapter) selectAndAddMailServer() (string, error) {
	if a.selectedMailServerEnode != "" {
		return a.selectedMailServerEnode, nil
	}

	config := a.node.Config()
	enode := randomItem(config.ClusterConfig.TrustedMailServers)
	errCh := waitForPeerAsync(
		a.node.GethNode().Server(),
		enode,
		p2p.PeerEventTypeAdd,
		time.Second*5,
	)

	log.Printf("[WhisperServiceAdapter::selectAndAddMailServer] randomly selected %s node", enode)

	if err := a.node.AddPeer(enode); err != nil {
		return "", err
	}

	err := <-errCh
	if err != nil {
		err = fmt.Errorf("failed to add mail server %s: %v", enode, err)
	} else {
		a.selectedMailServerEnode = enode
	}

	return enode, err
}

func (a *WhisperServiceAdapter) requestMessages(ctx context.Context, req shhext.MessagesRequest, followCursor bool) (resp shhext.MessagesResponse, err error) {
	shhextService, err := a.node.ShhExtService()
	if err != nil {
		return
	}

	shhextAPI := shhext.NewPublicAPI(shhextService)

	resp, err = shhextAPI.RequestMessagesSync(shhext.RetryConfig{
		BaseTimeout: time.Second * 10,
		StepTimeout: time.Second,
		MaxRetries:  3,
	}, req)
	if err != nil {
		return
	}

	log.Printf("[WhisperServiceAdapter::requestMessages] response = %+v, err = %v", resp, err)

	if resp.Error != nil {
		err = resp.Error
		return
	}

	if !followCursor || req.Cursor == "" {
		return
	}

	req.Cursor = resp.Cursor
	return a.requestMessages(ctx, req, true)
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
func (s whisperSubscription) Messages() ([]*protocol.Message, error) {
	f := s.shh.GetFilter(s.filterID)
	if f == nil {
		return nil, errors.New("filter does not exist")
	}

	items := f.Retrieve()
	result := make([]*protocol.Message, 0, len(items))

	for _, item := range items {
		decoded, err := protocol.DecodeMessage(item.Payload)
		if err != nil {
			log.Printf("failed to decode message: %v", err)
			continue
		}

		result = append(result, &protocol.Message{
			Decoded:   decoded,
			Hash:      item.EnvelopeHash.Bytes(),
			SigPubKey: item.SigToPubKey(),
		})
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

func createRichFilter(keys keysManager, options protocol.SubscribeOptions) (*whisper.Filter, error) {
	filter := whisper.Filter{
		PoW:      0,
		AllowP2P: true,
	}

	topic, err := topicForSubscribeOptions(options)
	if err != nil {
		return nil, err
	}
	filter.Topics = append(filter.Topics, topic[:])

	if options.Identity != nil {
		filter.KeyAsym = options.Identity
	}

	if options.ChatName != "" {
		symKeyID, err := keys.AddOrGetSymKeyFromPassword(options.ChatName)
		if err != nil {
			return nil, err
		}
		symKey, err := keys.GetRawSymKey(symKeyID)
		if err != nil {
			return nil, err
		}
		filter.KeySym = symKey
	}

	return &filter, nil
}
