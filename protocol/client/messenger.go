package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/messaging/multidevice"
)

type Messenger struct {
	identity *ecdsa.PrivateKey
	proto    protocol.Protocol
	db       Database

	mu sync.Mutex // guards public and private maps

	events *event.Feed
}

func NewMessenger(identity *ecdsa.PrivateKey, proto protocol.Protocol, db Database) *Messenger {
	events := &event.Feed{}
	return &Messenger{
		identity: identity,
		proto:    proto,
		db:       NewDatabaseWithEvents(db, events),

		events: events,
	}
}

func contactToChatOptions(c Contact) protocol.ChatOptions {
	return protocol.ChatOptions{
		ChatName:  c.Name,
		Recipient: c.PublicKey,
	}
}

func (m *Messenger) Start() error {
	log.Printf("[Messenger::Start]")
	var chatOptions []protocol.ChatOptions

	m.mu.Lock()
	defer m.mu.Unlock()

	contacts, err := m.db.Contacts()
	if err != nil {
		return errors.Wrap(err, "unable to read contacts from database")
	}

	for i := range contacts {
		chatOptions = append(chatOptions, contactToChatOptions(contacts[i]))
	}

	if err := m.proto.LoadChats(context.Background(), chatOptions); err != nil {
		return err
	}

	log.Printf("[Messenger::Start] request messages from mail sever")
	go m.ProcessMessages()

	return m.RequestAll(context.Background(), true)
}

func (m *Messenger) Stop() {
	log.Printf("[Messenger::Stop]")

	m.mu.Lock()
	defer m.mu.Unlock()
}

func (m *Messenger) handleDirectMessage(chatType protocol.ChatOptions, message protocol.Message) error {
	contact, err := m.db.GetOneToOneChat(message.SigPubKey)
	if err != nil {
		return errors.Wrap(err, "could not fetch chat from database")
	}
	if contact == nil {
		contact = &Contact{
			Type:      ContactPrivate,
			State:     ContactNew,
			Name:      pubkeyToHex(message.SigPubKey), // TODO(dshulyak) replace with 3-word funny name
			PublicKey: message.SigPubKey,
			Topic:     DefaultPrivateTopic(),
		}

		err := m.db.SaveContacts([]Contact{*contact})
		if err != nil {
			return errors.Wrap(err, "can't save a new contact")
		}
	}

	_, err = m.db.SaveMessages(*contact, []*protocol.Message{&message})
	if err == ErrMsgAlreadyExist {
		log.Printf("Message already exists")
		return nil
	} else if err != nil {
		return errors.Wrap(err, "can't add a message")
	}

	return nil
}

func (m *Messenger) handlePublicMessage(chatType protocol.ChatOptions, message protocol.Message) error {
	contact, err := m.db.GetPublicChat(chatType.ChatName)
	if err != nil {
		return errors.Wrap(err, "error getting public chat")
	} else if contact == nil {
		return errors.Wrap(err, "no chat for this message, is that a deleted chat?")
	}
	_, err = m.db.SaveMessages(*contact, []*protocol.Message{&message})
	if err == ErrMsgAlreadyExist {
		log.Printf("Message already exists")
		return nil
	} else if err != nil {
		return errors.Wrap(err, "can't add a message")
	}

	return nil
}

func (m *Messenger) handleMessageType(chatType protocol.ChatOptions, message protocol.Message) error {
	// TODO: handle group chats messages
	if chatType.OneToOne {
		return m.handleDirectMessage(chatType, message)
	}
	return m.handlePublicMessage(chatType, message)
}

func (m *Messenger) handlePairInstallationMessageType(chatType protocol.ChatOptions, sm *protocol.StatusMessage, message protocol.PairInstallationMessage) error {
	if !isPubKeyEqual(sm.SigPubKey, &m.identity.PublicKey) {
		return errors.New("Not coming from our identity, ignoring")
	}

	metadata := &multidevice.InstallationMetadata{
		Name:       message.Name,
		FCMToken:   message.FCMToken,
		DeviceType: message.DeviceType,
	}
	return m.proto.SetInstallationMetadata(context.TODO(), message.InstallationID, metadata)
}

func (m *Messenger) processMessage(message *protocol.ReceivedMessages) {
	for _, sm := range message.Messages {
		publicKey := sm.SigPubKey
		if publicKey == nil {
			log.Printf("No public key, ignoring")
		}

		switch sm.Message.(type) {
		case protocol.Message:
			// TODO: this fields should be in any message type
			m1 := sm.Message.(protocol.Message)
			m1.ID = sm.ID
			m1.SigPubKey = sm.SigPubKey

			if err := m.handleMessageType(message.ChatOptions, m1); err != nil {
				log.Printf("failed handling message: %+v", err)
				continue
			}
		case protocol.PairInstallationMessage:
			m1 := sm.Message.(protocol.PairInstallationMessage)
			if err := m.handlePairInstallationMessageType(message.ChatOptions, sm, m1); err != nil {
				log.Printf("failed handling message: %+v", err)
				continue
			}
		}

	}
}

func (m *Messenger) ProcessMessages() {
	for {
		msg, more := <-m.proto.GetMessagesChan()
		if !more {
			return
		}
		m.processMessage(msg)
	}
}

func (m *Messenger) Join(ctx context.Context, c Contact) error {
	log.Printf("[Messenger::Join] Joining a chat with contact %#v", c)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.proto.LoadChats(context.Background(), []protocol.ChatOptions{contactToChatOptions(c)}); err != nil {
		return err
	}

	opts := protocol.DefaultRequestOptions()
	// NOTE(dshulyak) join ctx shouldn't have an impact on history timeout.
	if err := m.Request(context.Background(), c, opts); err != nil {
		return err
	}
	return m.db.UpdateHistories([]History{{Contact: c, Synced: opts.To}})
}

// Messages reads all messages from database.
func (m *Messenger) Messages(c Contact, offset int64) ([]*protocol.Message, error) {
	return m.db.NewMessages(c, offset)
}

func (m *Messenger) Request(ctx context.Context, c Contact, options protocol.RequestOptions) error {
	err := enhanceRequestOptions(c, &options)
	if err != nil {
		return err
	}
	return m.proto.Request(ctx, options)
}

func (m *Messenger) requestHistories(ctx context.Context, histories []History, opts protocol.RequestOptions) error {
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

func (m *Messenger) RequestAll(ctx context.Context, newest bool) error {
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

func (m *Messenger) Send(c Contact, data []byte) ([]byte, error) {
	// FIXME(dshulyak) sending must be locked by contact to prevent sending second msg with same clock
	clock, err := m.db.LastMessageClock(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read last message clock for contact")
	}
	var message protocol.Message

	switch c.Type {
	case ContactPublicRoom:
		message = protocol.CreatePublicTextMessage(data, clock, c.Name)
	case ContactPrivate:
		message = protocol.CreatePrivateTextMessage(data, clock, c.Name)
	default:
		return nil, fmt.Errorf("failed to send message: unsupported contact type")
	}

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message")
	}
	opts, err := createSendOptions(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare send options")
	}

	log.Printf("[Messenger::Send] sending message")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	hash, err := m.proto.Send(ctx, encodedMessage, opts)
	if err != nil {
		return nil, errors.Wrap(err, "can't send a message")
	}

	log.Printf("[Messenger::Send] sent message with hash %x", hash)

	message.ID = hash
	message.SigPubKey = &m.identity.PublicKey
	_, err = m.db.SaveMessages(c, []*protocol.Message{&message})
	switch err {
	case ErrMsgAlreadyExist:
		log.Printf("[Messenger::Send] message with ID %x already exists", message.ID)
		return hash, nil
	case nil:
		return hash, nil
	default:
		return nil, errors.Wrap(err, "failed to save the message")
	}
}

func (m *Messenger) RemoveContact(c Contact) error {
	return m.db.DeleteContact(c)
}

func (m *Messenger) AddContact(c Contact) error {
	return m.db.SaveContacts([]Contact{c})
}

func (m *Messenger) Contacts() ([]Contact, error) {
	return m.db.Contacts()
}

func (m *Messenger) Leave(c Contact) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.proto.RemoveChats(context.Background(), []protocol.ChatOptions{contactToChatOptions(c)}); err != nil {
		return err
	}

	return nil
}

func (m *Messenger) Subscribe(events chan Event) event.Subscription {
	return m.events.Subscribe(events)
}

func pubkeyToHex(key *ecdsa.PublicKey) string {
	buf := crypto.FromECDSAPub(key)
	return hexutil.Encode(buf)
}

// isPubKeyEqual checks that two public keys are equal
func isPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
