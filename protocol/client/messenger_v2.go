package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"

	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

func NewMessengerV2(identity *ecdsa.PrivateKey, proto protocol.Protocol, db Database) MessengerV2 {
	feed := make(chan interface{})
	return MessengerV2{
		identity: identity,
		proto:    proto,
		db:       NewDatabaseWithEvents(db, feed),

		public: map[string]AsyncStream{},
		// FIXME(dshulyak) add sufficient buffer to this channel
		// it may block stream that receives messages
		events: feed,
	}
}

type MessengerV2 struct {
	identity *ecdsa.PrivateKey
	proto    protocol.Protocol
	db       Database

	mu      sync.Mutex
	public  map[string]AsyncStream
	private AsyncStream

	events chan interface{}
}

func NewMessanger(identity *ecdsa.PrivateKey, db Database, proto protocol.Protocol) MessengerV2 {
	return MessengerV2{
		identity: identity,
		db:       db,
		proto:    proto,

		public: map[string]AsyncStream{},
		events: make(chan interface{}),
	}
}

func (m *MessengerV2) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	contacts, err := m.db.Contacts()
	if err != nil {
		return errors.Wrap(err, "unable to read contacts from database")
	}
	private := []Contact{}
	for i := range contacts {
		if contacts[i].Type == ContactPublicKey {
			private = append(private, contacts[i])
		} else {
			stream := NewStream(context.Background(), contacts[i], m.proto, NewPublicHandler(contacts[i], m.db))
			err := stream.Start()
			if err != nil {
				return errors.Wrap(err, "unable to start stream")
			}
			m.public[contacts[i].Name] = stream
		}
	}
	// FIXME(dshulyak) even if we have no private contacts we still should start a stream for private messages.
	// this requires moving topic one level higher, from whisper adapter to the client
	if len(private) != 0 {
		any := private[0]
		stream := NewStream(context.Background(), any, m.proto, NewPrivateHandler(private, m.db))
		err := stream.Start()
		if err != nil {
			return errors.Wrap(err, "unable to start private stream")
		}
		m.private = stream
	}
	log.Printf("[INFO] request messages from mail sever")
	return m.RequestAll(context.Background(), true)
}

func (m *MessengerV2) Join(ctx context.Context, c Contact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.db.SaveContacts([]Contact{c})
	if err != nil {
		return errors.Wrap(err, "can't add contact to db")
	}
	_, exist := m.public[c.Name]
	if c.Type == ContactPublicKey || exist {
		// FIXME(dshulyak) don't request messages on every join
		// all messages must be requested in a single request when app starts
		return m.Request(ctx, c, protocol.DefaultRequestOptions())
	}
	log.Printf("[INFO] created stream for contact %s\n", c)
	stream := NewStream(context.Background(), c, m.proto, NewPublicHandler(c, m.db))
	err = stream.Start()
	if err != nil {
		return errors.Wrap(err, "can't subscribe to a stream")
	}
	m.public[c.Name] = stream
	log.Printf("[INFO] made request for new messages contact %s\n", c)
	return m.Request(ctx, c, protocol.DefaultRequestOptions())
}

// Messages reads all messages from database.
func (m *MessengerV2) Messages(c Contact, offset int64) ([]*protocol.Message, error) {
	return m.db.GetNewMessages(c, offset)
}

func (m *MessengerV2) Request(ctx context.Context, c Contact, options protocol.RequestOptions) error {
	err := enhanceRequestOptions(c, &options)
	if err != nil {
		return err
	}
	return m.proto.Request(ctx, options)
}

func (m *MessengerV2) RequestAll(ctx context.Context, newest bool) error {
	contacts, err := m.db.Contacts()
	if err != nil {
		return errors.Wrap(err, "error fetching contacts")
	}
	requestParams := protocol.DefaultRequestOptions()
	for _, c := range contacts {
		err = enhanceRequestOptions(c, &requestParams)
		if err != nil {
			return err
		}
	}
	return m.proto.Request(ctx, requestParams)
}

func (m *MessengerV2) Send(c Contact, data []byte) error {
	// FIXME(dshulyak) sending must be locked by contact to prevent sending second msg with same clock
	clock, err := m.db.LastMessageClock(c)
	if err != nil {
		return errors.Wrap(err, "failed to read last message clock for contact")
	}
	var message protocol.Message

	switch c.Type {
	case ContactPublicRoom:
		message = protocol.CreatePublicTextMessage(data, clock, c.Name)
	case ContactPublicKey:
		message = protocol.CreatePrivateTextMessage(data, clock, c.Name)
	default:
		return fmt.Errorf("failed to send message: unsupported contact type")
	}

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to encode message")
	}
	opts, err := createSendOptions(c)
	if err != nil {
		return errors.Wrap(err, "failed to prepare send options")
	}

	hash, err := m.proto.Send(context.Background(), encodedMessage, opts)
	if err != nil {
		return errors.Wrap(err, "can't send a message")
	}
	message.ID = hash
	message.SigPubKey = &m.identity.PublicKey
	_, err = m.db.SaveMessages(c, []*protocol.Message{&message})
	if err != nil {
		return errors.Wrap(err, "failed to save the message")
	}
	return nil
}

func (m *MessengerV2) RemoveContact(c Contact) error {
	return m.db.DeleteContact(c)
}

func (m *MessengerV2) AddContact(c Contact) error {
	return m.db.SaveContacts([]Contact{c})
}

func (m *MessengerV2) Contacts() ([]Contact, error) {
	return m.db.Contacts()
}

func (m *MessengerV2) Leave(c Contact) error {
	if c.Type == ContactPublicRoom {
		m.mu.Lock()
		defer m.mu.Unlock()
		stream, exist := m.public[c.Name]
		if !exist {
			return errors.New("stream doesn't exist")
		}
		stream.Stop()
		return nil
	}
	// TODO how to handle leave for private chat? block that peer?
	return nil
}

func (m *MessengerV2) Events() <-chan interface{} {
	return m.events
}
