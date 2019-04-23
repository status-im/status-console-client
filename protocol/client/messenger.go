package client

import (
	"crypto/ecdsa"
	"log"
	"sync"

	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

// Messenger coordinates chats.
type Messenger struct {
	sync.RWMutex

	proto    protocol.Protocol
	identity *ecdsa.PrivateKey
	db       *Database

	chats        map[Contact]*Chat
	chatsCancels map[Contact]chan struct{} // cancel handling chat events

	events chan interface{}
}

// NewMessenger returns a new Messanger.
func NewMessenger(proto protocol.Protocol, identity *ecdsa.PrivateKey, db *Database) *Messenger {
	return &Messenger{
		proto:        proto,
		identity:     identity,
		db:           db,
		chats:        make(map[Contact]*Chat),
		chatsCancels: make(map[Contact]chan struct{}),
		events:       make(chan interface{}),
	}
}

// Events returns a channel with chat events.
func (m *Messenger) Events() <-chan interface{} {
	m.RLock()
	defer m.RUnlock()
	return m.events
}

// Chat returns a chat for a given Contact.
func (m *Messenger) Chat(c Contact) *Chat {
	m.RLock()
	defer m.RUnlock()
	return m.chats[c]
}

// Join creates a new chat and creates a subscription.
func (m *Messenger) Join(contact Contact) (*Chat, error) {
	requestParams := protocol.DefaultRequestOptions()

	chat := m.Chat(contact)
	if chat != nil {
		return chat, chat.loadAndRequest(requestParams)
	}

	chat = NewChat(m.proto, m.identity, contact, m.db)
	cancel := make(chan struct{})

	m.Lock()
	m.chats[contact] = chat
	m.chatsCancels[contact] = cancel
	m.Unlock()

	go m.chatEventsLoop(chat, contact, cancel)

	if err := chat.subscribe(); err != nil {
		return chat, err
	}

	return chat, chat.loadAndRequest(requestParams)
}

func (m *Messenger) chatEventsLoop(chat *Chat, contact Contact, cancel chan struct{}) {
LOOP:
	for {
		select {
		case ev := <-chat.Events():
			log.Printf("[Messenger::Join] received an event: %+v", ev)
			m.events <- ev
		case <-chat.Done():
			log.Printf("[Messenger::Join] chat was left")
			break LOOP
		case <-cancel:
			break LOOP
		}
	}

	if err := chat.Err(); err != nil {
		m.events <- errorEvent{
			baseEvent: baseEvent{contact: contact, typ: EventTypeError},
			err:       err,
		}
	}

	chat.leave()

	m.Lock()
	delete(m.chats, contact)
	delete(m.chatsCancels, contact)
	m.Unlock()
}

// Leave unsubscribes from the chat.
func (m *Messenger) Leave(contact Contact) error {
	m.RLock()
	chat, ok := m.chats[contact]
	cancel := m.chatsCancels[contact]
	m.RUnlock()
	if !ok {
		return errors.New("chat for the contact not found")
	}

	chat.leave() // close chat loops; must happen before closing the chat events loop

	close(cancel) // close a loop reading and forwarding the chat events

	m.Lock()
	delete(m.chats, contact)
	delete(m.chatsCancels, contact)
	m.Unlock()

	return nil
}

func (m *Messenger) RequestAll(newest bool) error {
	var finalOpts protocol.RequestOptions

	for _, chat := range m.chats {
		opts, err := chat.RequestOptions(newest)
		if err != nil {
			return err
		}

		finalOpts.Chats = append(finalOpts.Chats, opts.Chats...)

		if opts.Limit > finalOpts.Limit {
			finalOpts.Limit = opts.Limit
		}
		if opts.From < finalOpts.From {
			finalOpts.From = opts.From
		}
		if opts.To > finalOpts.To {
			finalOpts.To = opts.To
		}
	}

	return m.proto.Request(nil, finalOpts)
}

// Contacts returns a list of contacts.
func (m *Messenger) Contacts() ([]Contact, error) {
	return m.db.Contacts()
}

// AddContact adds a new Contact. It detects duplicate entries.
func (m *Messenger) AddContact(c Contact) error {
	contacts, err := m.db.Contacts()
	if err != nil {
		return err
	}

	if ContainsContact(contacts, c) {
		return errors.New("duplicated contact")
	}

	contacts = append(contacts, c)
	return m.db.SaveContacts(contacts)
}

// RemoveContact removes a contact from the list.
func (m *Messenger) RemoveContact(c Contact) error {
	contacts, err := m.db.Contacts()
	if err != nil {
		return err
	}

	for i, item := range contacts {
		if item != c {
			continue
		}

		copy(contacts[i:], contacts[i+1:])
		contacts[len(contacts)-1] = Contact{}
		contacts = contacts[:len(contacts)-1]

		break
	}

	contacts = append(contacts, c)
	return m.db.SaveContacts(contacts)
}
