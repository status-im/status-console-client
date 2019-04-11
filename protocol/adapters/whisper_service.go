package adapters

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/shhext/chat"
	whisper "github.com/status-im/whisper/whisperv6"
)

type whisperServiceKeysManager struct {
	shh *whisper.Whisper

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *whisperServiceKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.shh.AddKeyPair(priv)
}

func (m *whisperServiceKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymKeyMutex.Lock()
	defer m.passToSymKeyMutex.Unlock()

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

	// PFS supports only one private key which should be provided
	// during PFS initialization.
	pfsPrivateKey *ecdsa.PrivateKey
	pfs           *chat.ProtocolService

	selectedMailServerEnode string
}

// WhisperServiceAdapter must implement Protocol interface.
var _ protocol.Protocol = (*WhisperServiceAdapter)(nil)

// NewWhisperServiceAdapter returns a new WhisperServiceAdapter.
func NewWhisperServiceAdapter(node *node.StatusNode, shh *whisper.Whisper) *WhisperServiceAdapter {
	return &WhisperServiceAdapter{
		node: node,
		shh:  shh,
		keysManager: &whisperServiceKeysManager{
			shh:               shh,
			passToSymKeyCache: make(map[string]string),
		},
	}
}

// InitPFS adds support for PFS messages.
func (a *WhisperServiceAdapter) InitPFS(baseDir string, privateKey *ecdsa.PrivateKey) error {
	addBundlesHandler := func(addedBundles []chat.IdentityAndIDPair) {
		log.Printf("added bundles: %v", addedBundles)
	}

	const (
		// TODO: manage these values properly
		dbPath        = "pfs_v1.db"
		sqlSecretKey  = "enc-key-abc"
		instalationID = "instalation-1"
	)

	dir := filepath.Join(baseDir, dbPath)
	persistence, err := chat.NewSQLLitePersistence(dir, sqlSecretKey)
	if err != nil {
		return err
	}

	a.pfsPrivateKey = privateKey
	a.pfs = chat.NewProtocolService(
		chat.NewEncryptionService(
			persistence,
			chat.DefaultEncryptionServiceConfig(instalationID),
		),
		addBundlesHandler,
	)

	return nil
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

	subWhisper := newWhisperSubscription(a.shh, a.pfs, a.pfsPrivateKey, filterID)
	sub := protocol.NewSubscription()

	go func() {
		defer subWhisper.Unsubscribe() // nolint: errcheck

		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				received, err := subWhisper.Messages()
				if err != nil {
					sub.Cancel(err)
					return
				}

				messages := a.handleMessages(received)
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

func (a *WhisperServiceAdapter) handleMessages(received []*whisper.ReceivedMessage) []*protocol.Message {
	var messages []*protocol.Message

	for _, item := range received {
		message, err := a.decodeMessage(item)
		if err != nil {
			log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
			continue
		}
		messages = append(messages, message)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Decoded.Clock < messages[j].Decoded.Clock
	})

	return messages
}

func (a *WhisperServiceAdapter) decodeMessage(message *whisper.ReceivedMessage) (*protocol.Message, error) {
	payload := message.Payload
	publicKey := message.SigToPubKey()
	hash := message.EnvelopeHash.Bytes()

	if a.pfs != nil {
		decryptedPayload, err := a.pfs.HandleMessage(a.pfsPrivateKey, publicKey, payload, hash)
		if err != nil {
			log.Printf("failed to handle message %#+x by PFS: %v", hash, err)
		} else {
			payload = decryptedPayload
		}
	}

	decoded, err := protocol.DecodeMessage(payload)
	if err != nil {
		return nil, err
	}

	return &protocol.Message{
		Decoded:   decoded,
		Hash:      hash,
		SigPubKey: publicKey,
	}, nil
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

	if a.pfs != nil {
		encryptedPayload, err := a.pfs.BuildDirectMessage(a.pfsPrivateKey, options.Recipient, data)
		if err != nil {
			return nil, err
		}
		data = encryptedPayload
	}

	message, err := createRichWhisperNewMessage(a.keysManager, data, options)
	if err != nil {
		return nil, err
	}

	// Only public Whisper API implements logic to send messages.
	shhAPI := whisper.NewPublicWhisperAPI(a.shh)
	return shhAPI.Post(ctx, message)
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
	shh          *whisper.Whisper
	pfs          *chat.ProtocolService
	myPrivateKey *ecdsa.PrivateKey
	filterID     string
}

// newWhisperSubscription returns a new whisperSubscription.
func newWhisperSubscription(shh *whisper.Whisper, pfs *chat.ProtocolService, pk *ecdsa.PrivateKey, filterID string) *whisperSubscription {
	return &whisperSubscription{
		shh:          shh,
		pfs:          pfs,
		myPrivateKey: pk,
		filterID:     filterID,
	}
}

// Messages retrieves a list of messages for a given filter.
func (s whisperSubscription) Messages() ([]*whisper.ReceivedMessage, error) {
	f := s.shh.GetFilter(s.filterID)
	if f == nil {
		return nil, errors.New("filter does not exist")
	}
	messages := f.Retrieve()
	return messages, nil
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
