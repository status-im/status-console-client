package messenger

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"sort"

	"github.com/pkg/errors"

	"github.com/status-im/status-console-client/protocol/v1"
)

const (
	EventTypeMessage int = iota + 1
	EventTypeError
)

type Event interface {
	Contact() Contact
	Type() int
}

type EventError interface {
	Event
	Err() error
}

type baseEvent struct {
	contact Contact
	typ     int
}

func (e baseEvent) Contact() Contact { return e.contact }
func (e baseEvent) Type() int        { return e.typ }

type eventError struct {
	baseEvent
	err error
}

func (e eventError) Err() error { return e.err }

type Chat struct {
	proto    protocol.Chat
	identity *ecdsa.PrivateKey

	contact Contact

	db *Database

	sub    *protocol.Subscription
	events chan interface{}
	err    error

	lastClock int64

	messages       []*protocol.ReceivedMessage // ordered by Clock
	messagesByHash map[string]*protocol.ReceivedMessage
}

func NewChat(proto protocol.Chat, identity *ecdsa.PrivateKey, c Contact, db *Database) *Chat {
	return &Chat{
		proto:          proto,
		identity:       identity,
		contact:        c,
		db:             db,
		messagesByHash: make(map[string]*protocol.ReceivedMessage),
	}
}

func (c *Chat) Events() <-chan interface{} {
	return c.events
}

func (c *Chat) Err() error {
	return c.err
}

func (c *Chat) Messages() []*protocol.ReceivedMessage {
	return c.messages
}

func (c *Chat) Subscribe() error {
	opts := protocol.SubscribeOptions{}
	if c.contact.Type == ContactPublicChat {
		opts.ChatName = c.contact.Name
	} else {
		opts.Identity = c.identity
	}

	messages := make(chan *protocol.ReceivedMessage)

	sub, err := c.proto.Subscribe(context.Background(), messages, opts)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe")
	}

	c.events = make(chan interface{})

	go c.readLoop(messages, sub)

	params := protocol.DefaultRequestMessagesParams()

	cachedMessages, err := c.db.Messages(
		c.contact,
		params.From,
		params.To,
	)
	if err != nil {
		return errors.Wrap(err, "db failed to get messages")
	}

	go func() {
		for _, m := range cachedMessages {
			messages <- m
		}
	}()

	if err := c.proto.Request(context.Background(), params); err != nil {
		return errors.Wrap(err, "failed to request for messages")
	}

	return nil
}

func (c *Chat) Unsubscribe() {
	if c.sub == nil {
		return
	}
	c.sub.Unsubscribe()
}

func (c *Chat) readLoop(messages <-chan *protocol.ReceivedMessage, sub *protocol.Subscription) {
	defer close(c.events)

	for {
		select {
		case m := <-messages:
			if err := c.handleMessage(m); err != nil {
				c.err = err
				return
			}
			c.events <- baseEvent{contact: c.contact, typ: EventTypeMessage}
		case <-sub.Done():
			c.err = sub.Err()
			return
		}
	}
}

func (c *Chat) handleMessage(message *protocol.ReceivedMessage) error {
	lessFn := func(i, j int) bool {
		return c.messages[i].Decoded.Clock < c.messages[j].Decoded.Clock
	}
	hash := hex.EncodeToString(message.Hash)

	// the message already exists
	if _, ok := c.messagesByHash[hash]; ok {
		return nil
	}

	c.updateLastClock(message)

	c.messagesByHash[hash] = message
	c.messages = append(c.messages, message)

	isSorted := sort.SliceIsSorted(c.messages, lessFn)
	if !isSorted {
		sort.Slice(c.messages, lessFn)
	}

	if err := c.db.SaveMessages(c.contact, message); err != nil {
		return err
	}

	return nil
}

func (c *Chat) updateLastClock(m *protocol.ReceivedMessage) {
	if m.Decoded.Clock > c.lastClock {
		c.lastClock = m.Decoded.Clock
	}
}
