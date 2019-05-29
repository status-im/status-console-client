package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

func NewMessengerV2(identity *ecdsa.PrivateKey, proto protocol.Protocol, db Database) MessengerV2 {
	events := &event.Feed{}
	return MessengerV2{
		identity: identity,
		proto:    proto,
		db:       NewDatabaseWithEvents(db, events),

		public:  make(map[string]*Stream),
		private: make(map[string]*Stream),
		events:  events,
	}
}

type MessengerV2 struct {
	identity *ecdsa.PrivateKey
	proto    protocol.Protocol
	db       Database

	mu      sync.Mutex
	public  map[string]*Stream
	private map[string]*Stream

	events *event.Feed
}

func (m *MessengerV2) Start() error {
	log.Printf("[Messenger::Start]")

	m.mu.Lock()
	defer m.mu.Unlock()

	contacts, err := m.db.Contacts()
	if err != nil {
		return errors.Wrap(err, "unable to read contacts from database")
	}

	// For each contact, a new Stream is created that subscribes to the protocol
	// and forwards incoming messages to the Stream instance.
	// We iterate over all contacts, however, there are two cases:
	// (1) Public chats where each has a unique topic,
	// (2) Private chats where a single or shared topic is used.
	// This means that we don't know from which contact the message
	// came from until it is examined.
	for i := range contacts {
		options, err := createSubscribeOptions(contacts[i])
		if err != nil {
			return err
		}

		// For contacts with public key, we just need to make sure
		// each possible topic has a stream. For a single topic
		// for all private conversations, the map will have a len of 1.
		// In the future, private conversations will have sharded topics,
		// which means there will be many conversation over a particular topic
		// but there will be more than one topic.
		if contacts[i].Type == ContactPublicKey {
			_, exist := m.private[contacts[i].Topic]

			if exist {
				continue
			}

			stream := NewStream(m.proto, StreamHandlerMultiplexed(m.db))
			if err := stream.Start(context.Background(), options); err != nil {
				return errors.Wrap(err, "unable to start private stream")
			}

			m.private[contacts[i].Topic] = stream
		} else {
			_, exist := m.public[contacts[i].Topic]

			if exist {
				return fmt.Errorf("multiple public chats with same topic: %s", contacts[i].Topic)
			}

			stream := NewStream(m.proto, StreamHandlerForContact(contacts[i], m.db))
			if err := stream.Start(context.Background(), options); err != nil {
				return errors.Wrap(err, "unable to start stream")
			}

			m.public[contacts[i].Topic] = stream
		}
	}

	log.Printf("[Messenger::Start] request messages from mail sever")

	return m.RequestAll(context.Background(), true)
}

func (m *MessengerV2) Join(ctx context.Context, c Contact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.Type == ContactPublicKey {
		return m.joinPrivate(ctx, c)
	}
	return m.joinPublic(ctx, c)
}

func (m *MessengerV2) joinPrivate(ctx context.Context, c Contact) error {
	_, exist := m.private[c.Topic]
	if exist {
		return nil
	}

	subOpts, err := createSubscribeOptions(c)
	if err != nil {
		return errors.Wrap(err, "failed to create SubscribeOptions")
	}

	stream := NewStream(m.proto, StreamHandlerMultiplexed(m.db))
	if err := stream.Start(context.Background(), subOpts); err != nil {
		return errors.Wrap(err, "can't subscribe to a stream")
	}
	m.private[c.Name] = stream

	opts := protocol.DefaultRequestOptions()
	if err := m.Request(ctx, c, opts); err != nil {
		return err
	}
	return m.db.UpdateHistories([]History{{Contact: c, Synced: opts.To}})
}

func (m *MessengerV2) joinPublic(ctx context.Context, c Contact) error {
	_, exist := m.public[c.Topic]
	if exist {
		// FIXME(dshulyak) don't request messages on every join
		// all messages must be requested in a single request when app starts
		return nil
	}

	subOpts, err := createSubscribeOptions(c)
	if err != nil {
		return err
	}

	stream := NewStream(m.proto, StreamHandlerForContact(c, m.db))
	if err := stream.Start(context.Background(), subOpts); err != nil {
		return errors.Wrap(err, "can't subscribe to a stream")
	}
	m.public[c.Name] = stream

	opts := protocol.DefaultRequestOptions()
	if err := m.Request(ctx, c, opts); err != nil {
		return err
	}
	return m.db.UpdateHistories([]History{{Contact: c, Synced: opts.To}})
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

func (m *MessengerV2) requestHistories(ctx context.Context, histories []History, opts protocol.RequestOptions) error {
	log.Printf("[Messenger::requestHistories] requesting messages for chats %+v: from %d to %d\n", opts.Chats, opts.From, opts.To)

	start := time.Now()

	err := m.proto.Request(ctx, opts)
	if err != nil {
		return err
	}

	log.Printf("[Messenger::requestHistories] requesting message for chats %+v finished in %s\n", opts.Chats, time.Since(start))

	for i := range histories {
		histories[i].Synced = opts.To
	}
	return m.db.UpdateHistories(histories)
}

func (m *MessengerV2) RequestAll(ctx context.Context, newest bool) error {
	// FIXME(dshulyak) if newest is false request 24 hour of messages older then the
	// earliest envelope for each contact.
	histories, err := m.db.Histories()
	if err != nil {
		return errors.Wrap(err, "error fetching contacts")
	}
	var (
		now               = time.Now()
		synced, notsynced = splitIntoSyncedNotSynced(histories)
		errors            = make(chan error, 2)
		wg                sync.WaitGroup
	)
	if len(synced) != 0 {
		wg.Add(1)
		go func() {
			errors <- m.requestHistories(ctx, synced, syncedToOpts(synced, now))
			wg.Done()
		}()
	}
	if len(notsynced) != 0 {
		wg.Add(1)
		go func() {
			errors <- m.requestHistories(ctx, notsynced, notsyncedToOpts(notsynced, now))
			wg.Done()
		}()
	}
	wg.Wait()

	log.Printf("[Messenger::RequestAll] finished requesting histories")

	close(errors)
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
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

	log.Printf("[Messenger::Send] sending message")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	hash, err := m.proto.Send(ctx, encodedMessage, opts)
	if err != nil {
		return errors.Wrap(err, "can't send a message")
	}

	log.Printf("[Messenger::Send] sent message with hash %x", hash)

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

func (m *MessengerV2) Subscribe(events chan Event) event.Subscription {
	return m.events.Subscribe(events)
}
