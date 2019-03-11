package client

import (
	"crypto/ecdsa"
	"sync"

	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

// Messenger coordinates chats.
type Messenger struct {
	proto    protocol.Chat
	identity *ecdsa.PrivateKey
	db       *Database

	chats map[Contact]*Chat
	wg    sync.WaitGroup

	events chan interface{}
}

// NewMessanger returns a new Messanger.
func NewMessenger(proto protocol.Chat, identity *ecdsa.PrivateKey, db *Database) *Messenger {
	return &Messenger{
		proto:    proto,
		identity: identity,
		db:       db,
		chats:    make(map[Contact]*Chat),
		events:   make(chan interface{}),
	}
}

// Events returns a channel with chat events.
func (m *Messenger) Events() <-chan interface{} {
	return m.events
}

// Join creates a new chat and creates a subscription.
func (m *Messenger) Join(contact Contact) error {
	chat := NewChat(m.proto, m.identity, contact, m.db)

	if err := chat.Subscribe(); err != nil {
		return err
	}

	m.chats[contact] = chat

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		for ev := range chat.Events() {
			m.events <- ev
		}

		if err := chat.Err(); err != nil {
			m.events <- baseEvent{contact: contact, typ: EventTypeError}
		}
	}()

	return nil
}

// Leave unsubscribes from the chat.
func (m *Messenger) Leave(contact Contact) error {
	chat, ok := m.chats[contact]
	if !ok {
		return errors.New("chat for the contact not found")
	}

	chat.Unsubscribe()
	delete(m.chats, contact)

	return nil
}

// Messages returns a list of messages for a given contact.
func (m *Messenger) Messages(contact Contact) ([]*protocol.Message, error) {
	chat, ok := m.chats[contact]
	if !ok {
		return nil, errors.New("chat for the contact not found")
	}

	return chat.Messages(), nil
}

func (m *Messenger) Request(contact Contact, params protocol.RequestOptions) error {
	chat, ok := m.chats[contact]
	if !ok {
		return errors.New("chat for the contact not found")
	}

	return chat.Request(params)
}

func (m *Messenger) Send(contact Contact, data []byte) error {
	chat, ok := m.chats[contact]
	if !ok {
		return errors.New("chat for the contact not found")
	}

	return chat.Send(data)
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
		if item == c {
			copy(contacts[i:], contacts[i+1:])
			contacts[len(contacts)-1] = Contact{}
			contacts = contacts[:len(contacts)-1]
		}
	}

	contacts = append(contacts, c)

	return m.db.SaveContacts(contacts)
}
