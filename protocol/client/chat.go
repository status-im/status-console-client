package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/status-im/status-console-client/protocol/v1"
)

// Chat represents a single conversation either public or private.
// It subscribes for messages and allows to send messages.
type Chat struct {
	sync.RWMutex

	proto protocol.Chat

	// Identity and Contact between the conversation happens.
	identity *ecdsa.PrivateKey
	contact  Contact

	db *Database

	sub    *protocol.Subscription
	events chan interface{}
	err    error

	lastClock int64

	ownMessages chan *protocol.Message // my private messages channel
	// TODO: make it a ring buffer
	messages       []*protocol.Message          // all messages ordered by Clock
	messagesByHash map[string]*protocol.Message // quick access to messages by hash
}

// NewChat returns a new Chat instance.
func NewChat(proto protocol.Chat, identity *ecdsa.PrivateKey, c Contact, db *Database) *Chat {
	return &Chat{
		proto:          proto,
		identity:       identity,
		contact:        c,
		db:             db,
		events:         make(chan interface{}),
		ownMessages:    make(chan *protocol.Message),
		messagesByHash: make(map[string]*protocol.Message),
	}
}

// Events returns a channel with Chat events.
func (c *Chat) Events() <-chan interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.events
}

// Err returns a cached error.
func (c *Chat) Err() error {
	c.RLock()
	defer c.RUnlock()
	return c.err
}

// Messages return a list of currently cached messages.
func (c *Chat) Messages() []*protocol.Message {
	c.RLock()
	defer c.RUnlock()
	return c.messages
}

// Subscribe reads messages from the network.
// TODO: change method name to Join().
func (c *Chat) Subscribe() (err error) {
	c.RLock()
	sub := c.sub
	c.RUnlock()

	if sub != nil {
		err = errors.New("already subscribed")
		return
	}

	opts := protocol.SubscribeOptions{}
	if c.contact.Type == ContactPublicChat {
		opts.ChatName = c.contact.Name
	} else {
		opts.Identity = c.identity
	}

	messages := make(chan *protocol.Message)

	sub, err = c.proto.Subscribe(context.Background(), messages, opts)
	if err != nil {
		err = errors.Wrap(err, "failed to subscribe")
		return
	}

	c.Lock()
	c.sub = sub
	c.Unlock()

	cancel := make(chan struct{}) // can be closed by any loop

	go c.readLoop(messages, sub, cancel)
	go c.readOwnMessagesLoop(c.ownMessages, cancel)

	// Load should have it's own lock.
	return c.Load()
}

// Load loads messages from the database cache and the network.
func (c *Chat) Load() error {
	params := protocol.DefaultRequestOptions()

	// Get already cached messages from the database.
	cachedMessages, err := c.db.Messages(
		c.contact,
		params.From,
		params.To,
	)
	if err != nil {
		return errors.Wrap(err, "db failed to get messages")
	}

	c.Lock()
	c.handleMessages(cachedMessages...)
	c.Unlock()

	go func() {
		log.Printf("[Chat::Subscribe] sending EventTypeInit")
		c.events <- baseEvent{contact: c.contact, typ: EventTypeInit}
		log.Printf("[Chat::Subscribe] sent EventTypeInit")
	}()

	if c.contact.Type == ContactPublicChat {
		params.ChatName = c.contact.Name
	} else {
		params.Recipient = c.contact.PublicKey
	}
	// Request historic messages from the network.
	if err := c.request(params); err != nil {
		return errors.Wrap(err, "failed to request for messages")
	}

	return nil
}

// Unsubscribe cancels the current subscription.
func (c *Chat) Unsubscribe() {
	c.RLock()
	defer c.RUnlock()
	if c.sub != nil {
		c.sub.Unsubscribe()
	}
}

// Request sends a request for historic messages.
func (c *Chat) Request(params protocol.RequestOptions) error {
	return c.request(params)
}

func (c *Chat) request(params protocol.RequestOptions) error {
	return c.proto.Request(context.Background(), params)
}

// Send sends a message into the network.
func (c *Chat) Send(data []byte) error {
	var messageType string

	text := strings.TrimSpace(string(data))
	ts := time.Now().Unix() * 1000
	clock := protocol.CalcMessageClock(c.lastClock, ts)
	opts := protocol.SendOptions{
		Identity: c.identity,
	}

	if c.contact.Type == ContactPublicChat {
		opts.ChatName = c.contact.Name
		messageType = protocol.MessageTypePublicGroupUserMessage
	} else {
		opts.Recipient = c.contact.PublicKey
		messageType = protocol.MessageTypePrivateUserMessage
	}

	message := protocol.StatusMessage{
		Text:      text,
		ContentT:  protocol.ContentTypeTextPlain,
		MessageT:  messageType,
		Clock:     clock,
		Timestamp: ts,
		Content:   protocol.StatusMessageContent{ChatID: c.contact.Name, Text: text},
	}
	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to encode message")
	}

	c.Lock()
	c.updateLastClock(clock)
	c.Unlock()

	hash, err := c.proto.Send(context.Background(), encodedMessage, opts)

	// Own messages need to be pushed manually to the pipeline.
	if c.contact.Type == ContactPrivateChat {
		log.Printf("[Chat::Send] sent a private message")

		c.ownMessages <- &protocol.Message{
			Decoded:   message,
			SigPubKey: &c.identity.PublicKey,
			Hash:      hash,
		}
	}

	return err
}

func (c *Chat) readLoop(messages <-chan *protocol.Message, sub *protocol.Subscription, cancel chan struct{}) {
	defer close(cancel)

	for {
		select {
		case m := <-messages:
			if c.HasMessage(m) {
				break
			}

			c.Lock()
			c.handleMessages(m)
			c.Unlock()

			if err := c.saveMessages(m); err != nil {
				c.Lock()
				c.err = err
				c.Unlock()
				return
			}

			c.events <- messageEvent{
				baseEvent: baseEvent{
					contact: c.contact,
					typ:     EventTypeMessage,
				},
				message: m,
			}
		case <-sub.Done():
			c.err = sub.Err()
			return
		case <-cancel:
			return
		}
	}
}

func (c *Chat) readOwnMessagesLoop(messages <-chan *protocol.Message, cancel chan struct{}) {
	defer close(cancel)

	for {
		select {
		case m := <-messages:
			if c.HasMessage(m) {
				break
			}

			c.Lock()
			c.handleMessages(m)
			c.Unlock()

			if err := c.saveMessages(m); err != nil {
				c.Lock()
				c.err = err
				c.Unlock()
				return
			}

			c.events <- messageEvent{
				baseEvent: baseEvent{
					contact: c.contact,
					typ:     EventTypeMessage,
				},
				message: m,
			}
		case <-cancel:
			return
		}
	}
}

func (c *Chat) handleMessages(messages ...*protocol.Message) {
	for _, message := range messages {
		c.updateLastClock(message.Decoded.Clock)

		hash := messageHashStr(message)

		c.messagesByHash[hash] = message
		c.messages = append(c.messages, message)

		sort.Slice(c.messages, c.lessFn)
	}
}

func (c *Chat) saveMessages(messages ...*protocol.Message) error {
	return c.db.SaveMessages(c.contact, messages)
}

func (c *Chat) lessFn(i, j int) bool {
	return c.messages[i].Decoded.Clock < c.messages[j].Decoded.Clock
}

func (c *Chat) updateLastClock(clock int64) {
	if clock > c.lastClock {
		c.lastClock = clock
	}
}

func (c *Chat) hasMessage(m *protocol.Message) bool {
	hash := messageHashStr(m)
	_, ok := c.messagesByHash[hash]
	return ok
}

// HasMessage returns true if a given message is already cached.
func (c *Chat) HasMessage(m *protocol.Message) bool {
	c.Lock()
	defer c.Unlock()
	return c.hasMessage(m)
}

func messageHashStr(m *protocol.Message) string {
	return hex.EncodeToString(m.Hash)
}
