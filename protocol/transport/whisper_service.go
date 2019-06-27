package transport

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-console-client/protocol/subscription"
)

var (
	// ErrNoMailservers returned if there is no configured mailservers that can be used.
	ErrNoMailservers = errors.New("no configured mailservers")
)

type WhisperServiceKeysManager struct {
	shh *whisper.Whisper

	// Identity of the current user.
	privateKey *ecdsa.PrivateKey

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *WhisperServiceKeysManager) PrivateKey() *ecdsa.PrivateKey {
	return m.privateKey
}

func (m *WhisperServiceKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.shh.AddKeyPair(priv)
}

func (m *WhisperServiceKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
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

func (m *WhisperServiceKeysManager) GetRawSymKey(id string) ([]byte, error) {
	return m.shh.GetSymKey(id)
}

type server interface {
	Connected(enode.ID) (bool, error)
	AddPeer(string) error
}

// WhisperServiceTransport is a transport based on Whisper service.
type WhisperServiceTransport struct {
	node        server
	shh         *whisper.Whisper
	shhextAPI   *shhext.PublicAPI
	keysManager *WhisperServiceKeysManager

	mailservers             []string
	selectedMailServerEnode string
}

var _ WhisperTransport = (*WhisperServiceTransport)(nil)

// NewWhisperService returns a new WhisperServiceTransport.
func NewWhisperServiceTransport(
	node server,
	mailservers []string,
	shh *whisper.Whisper,
	shhextService *shhext.Service,
	privateKey *ecdsa.PrivateKey,
) *WhisperServiceTransport {
	return &WhisperServiceTransport{
		node:        node,
		shh:         shh,
		mailservers: mailservers,
		shhextAPI:   shhext.NewPublicAPI(shhextService),
		keysManager: &WhisperServiceKeysManager{
			shh:               shh,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
	}
}

func (a *WhisperServiceTransport) KeysManager() *WhisperServiceKeysManager {
	return a.keysManager
}

// Subscribe subscribes to a public chat using the Whisper service.
func (a *WhisperServiceTransport) Subscribe(
	ctx context.Context,
	in chan<- *whisper.ReceivedMessage,
	filter *whisper.Filter,
) (*subscription.Subscription, error) {
	filterID, err := a.shh.Subscribe(filter)
	if err != nil {
		return nil, err
	}

	subWhisper := newWhisperSubscription(a.shh, filterID)
	sub := subscription.New()

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

				for _, message := range received {
					in <- message
				}
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

// Send sends a new message using the Whisper service.
func (a *WhisperServiceTransport) Send(ctx context.Context, newMessage whisper.NewMessage) ([]byte, error) {
	// Only public Whisper API implements logic to send messages.
	shhAPI := whisper.NewPublicWhisperAPI(a.shh)
	return shhAPI.Post(ctx, newMessage)
}

type RequestOptions struct {
	Topics   []whisper.TopicType
	Password string
	Limit    int
	From     int64 // in seconds
	To       int64 // in seconds
}

// Request requests messages from mail servers.
func (a *WhisperServiceTransport) Request(ctx context.Context, options RequestOptions) error {
	// TODO: remove from here. MailServerEnode must be provided in the params.
	enode, err := a.selectAndAddMailServer()
	if err != nil {
		return err
	}

	keyID, err := a.keysManager.AddOrGetSymKeyFromPassword(options.Password)
	if err != nil {
		return err
	}

	req, err := createShhextRequestMessagesParam(enode, keyID, options)
	if err != nil {
		return err
	}

	_, err = a.requestMessages(ctx, req, true)
	return err
}

func (a *WhisperServiceTransport) selectAndAddMailServer() (string, error) {
	var enodeAddr string
	if a.selectedMailServerEnode != "" {
		enodeAddr = a.selectedMailServerEnode
	} else {
		if len(a.mailservers) == 0 {
			return "", ErrNoMailservers
		}
		enodeAddr = randomItem(a.mailservers)
	}
	log.Printf("dialing mail server %s", enodeAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err := Dial(ctx, a.node, enodeAddr, DialOpts{PollInterval: 200 * time.Millisecond})
	cancel()
	if err == nil {
		a.selectedMailServerEnode = enodeAddr
		return enodeAddr, nil
	}
	return "", fmt.Errorf("peer %s failed to connect: %v", enodeAddr, err)
}

func (a *WhisperServiceTransport) requestMessages(ctx context.Context, req shhext.MessagesRequest, followCursor bool) (resp shhext.MessagesResponse, err error) {
	log.Printf("[WhisperServiceTransport::requestMessages] request for a chunk with %d messages", req.Limit)

	start := time.Now()
	resp, err = a.shhextAPI.RequestMessagesSync(shhext.RetryConfig{
		BaseTimeout: time.Second * 10,
		StepTimeout: time.Second,
		MaxRetries:  3,
	}, req)
	if err != nil {
		log.Printf("[WhisperServiceTransport::requestMessages] failed with err: %v", err)
		return
	}

	log.Printf("[WhisperServiceTransport::requestMessages] delivery of %d message took %v", req.Limit, time.Since(start))
	log.Printf("[WhisperServiceTransport::requestMessages] response: %+v", resp)

	if resp.Error != nil {
		err = resp.Error
		return
	}
	if !followCursor || resp.Cursor == "" {
		return
	}

	req.Cursor = resp.Cursor
	log.Printf("[WhisperServiceTransport::requestMessages] request messages with cursor %v", req.Cursor)
	return a.requestMessages(ctx, req, true)
}

// whisperSubscription encapsulates a Whisper filter.
type whisperSubscription struct {
	shh      *whisper.Whisper
	filterID string
}

// newWhisperSubscription returns a new whisperSubscription.
func newWhisperSubscription(shh *whisper.Whisper, filterID string) *whisperSubscription {
	return &whisperSubscription{
		shh:      shh,
		filterID: filterID,
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

func createShhextRequestMessagesParam(enode, symKeyID string, options RequestOptions) (shhext.MessagesRequest, error) {
	req := shhext.MessagesRequest{
		MailServerPeer: enode,
		From:           uint32(options.From),  // TODO: change to int in status-go
		To:             uint32(options.To),    // TODO: change to int in status-go
		Limit:          uint32(options.Limit), // TODO: change to int in status-go
		SymKeyID:       symKeyID,
		Topics:         options.Topics,
	}

	return req, nil
}
