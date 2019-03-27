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

	proto    protocol.Chat
	identity *ecdsa.PrivateKey
	db       *Database

	chats        map[Contact]*Chat
	chatsCancels map[Contact]chan struct{} // cancel handling chat events

	events chan interface{}
}

// NewMessenger returns a new Messanger.
func NewMessenger(proto protocol.Chat, identity *ecdsa.PrivateKey, db *Database) *Messenger {
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

func (m *Messenger) Chat(c Contact) *Chat {
	m.RLock()
	defer m.RUnlock()
	return m.chats[c]
}

// Join creates a new chat and creates a subscription.
func (m *Messenger) Join(contact Contact, params protocol.RequestOptions) error {
	chat := m.Chat(contact)
	if chat != nil {
		return chat.load(params)
	}

	chat = NewChat(m.proto, m.identity, contact, m.db)
	cancel := make(chan struct{})

	m.Lock()
	m.chats[contact] = chat
	m.chatsCancels[contact] = cancel
	m.Unlock()

	go m.chatEventsLoop(chat, contact, cancel)

	return chat.Subscribe(params)
}

func (m *Messenger) chatEventsLoop(chat *Chat, contact Contact, cancel chan struct{}) {
LOOP:
	for {
		select {
		case ev := <-chat.Events():
			log.Printf("[Messenger::Join] received an event: %+v", ev)
			m.events <- ev
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
	m.Unlock()

	return nil
}

func (m *Messenger) Contacts() ([]Contact, error) {
	return m.db.Contacts()
}

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
