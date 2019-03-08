package messenger

import (
	"crypto/ecdsa"
	"sync"

	"github.com/pkg/errors"

	"github.com/status-im/status-console-client/protocol/v1"
)

type Messenger struct {
	proto    protocol.Chat
	identity *ecdsa.PrivateKey
	db       *Database

	chats map[Contact]*Chat
	wg    sync.WaitGroup

	events chan interface{}
}

func NewMessenger(proto protocol.Chat, identity *ecdsa.PrivateKey, db *Database) *Messenger {
	return &Messenger{
		proto:    proto,
		identity: identity,
		db:       db,
		chats:    make(map[Contact]*Chat),
		events:   make(chan interface{}),
	}
}

func (m *Messenger) Events() <-chan interface{} {
	return m.events
}

func (m *Messenger) Join(contact Contact) error {
	chat := NewChat(m.proto, m.identity, contact, m.db)

	go func() {
		m.wg.Add(1)
		defer m.wg.Done()

		for ev := range chat.Events() {
			m.events <- ev
		}
		if err := chat.Err(); err != nil {
			m.events <- baseEvent{contact: contact, typ: EventTypeError}
		}
	}()

	if err := chat.Subscribe(); err != nil {
		return err
	}

	m.chats[contact] = chat

	return nil
}

func (m *Messenger) Leave(contact Contact) error {
	chat, ok := m.chats[contact]
	if !ok {
		return errors.New("chat for the contact not found")
	}

	chat.Unsubscribe()
	delete(m.chats, contact)

	return nil
}

func (m *Messenger) Messages(contact Contact) ([]*protocol.ReceivedMessage, error) {
	chat, ok := m.chats[contact]
	if !ok {
		return nil, errors.New("chat for the contact not found")
	}

	return chat.Messages(), nil
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
