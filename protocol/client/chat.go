package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
)

// Chat represents a single conversation either public or private.
// It subscribes for messages and allows to send messages.
type Chat struct {
	sync.RWMutex

	proto protocol.Protocol

	// Identity and Contact between the conversation happens.
	identity *ecdsa.PrivateKey
	contact  Contact

	db *Database

	sub    *protocol.Subscription
	events chan interface{}
	err    error

	cancel chan struct{} // can be closed by any goroutine and closes all others

	lastClock int64

	ownMessages chan *protocol.Message // my private messages channel
	// TODO: make it a ring buffer. It will require loading messages from the database.
	messages       []*protocol.Message          // all messages ordered by Clock
	messagesByHash map[string]*protocol.Message // quick access to messages by hash
}

// NewChat returns a new Chat instance.
func NewChat(proto protocol.Protocol, identity *ecdsa.PrivateKey, contact Contact, db *Database) *Chat {
	c := Chat{
		proto:          proto,
		identity:       identity,
		contact:        contact,
		db:             db,
		events:         make(chan interface{}),
		cancel:         make(chan struct{}),
		ownMessages:    make(chan *protocol.Message),
		messagesByHash: make(map[string]*protocol.Message),
	}

	go c.readOwnMessagesLoop(c.ownMessages, c.cancel)

	return &c
}

func (c *Chat) leave() {
	c.Lock()
	if c.sub != nil {
		c.sub.Unsubscribe()
	} else {
		close(c.cancel)
	}
	c.Unlock()
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

// HasMessage returns true if a given message is already cached.
func (c *Chat) HasMessage(m *protocol.Message) bool {
	c.RLock()
	defer c.RUnlock()
	return c.hasMessage(m)
}

func (c *Chat) hasMessage(m *protocol.Message) bool {
	hash := messageHashStr(m)
	_, ok := c.messagesByHash[hash]
	return ok
}

// Subscribe reads messages from the network.
//
// TODO: consider removing getting data from this method.
// Instead, getting data should be a separate call.
func (c *Chat) Subscribe(params protocol.RequestOptions) (err error) {
	c.RLock()
	sub := c.sub
	c.RUnlock()

	if sub != nil {
		return errors.New("already subscribed")
	}

	opts, err := extendSubscribeOptions(protocol.SubscribeOptions{}, c)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe")
	}

	messages := make(chan *protocol.Message)

	sub, err = c.proto.Subscribe(context.Background(), messages, opts)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe")
	}

	c.Lock()
	c.sub = sub
	c.Unlock()

	go c.readLoop(messages, sub, c.cancel)

	return c.load(params)
}

// Load loads messages from the database cache and the network.
func (c *Chat) load(options protocol.RequestOptions) error {
	// Get already cached messages from the database.
	cachedMessages, err := c.db.Messages(
		c.contact,
		options.From,
		options.To,
	)
	if err != nil {
		return errors.Wrap(err, "db failed to get messages")
	}

	c.handleMessages(cachedMessages...)

	go func() {
		log.Printf("[Chat::Subscribe] sending EventTypeInit")
		c.events <- baseEvent{contact: c.contact, typ: EventTypeInit}
		log.Printf("[Chat::Subscribe] sent EventTypeInit")
	}()

	// Request historic messages from the network.
	if err := c.request(options); err != nil {
		return errors.Wrap(err, "failed to request for messages")
	}

	return nil
}

func (c *Chat) request(options protocol.RequestOptions) error {
	opts, err := extendRequestOptions(options, c)
	if err != nil {
		return err
	}
	return c.proto.Request(context.Background(), opts)
}

// Request historic messages.
func (c *Chat) Request(options protocol.RequestOptions) error {
	return c.request(options)
}

// Send sends a message into the network.
func (c *Chat) Send(data []byte) error {
	// If cancel is closed then it will return an error.
	// Otherwise, the execution will continue.
	// This is needed to prevent sending messages
	// if the chat is already left/canceled
	// as a it can't be guaranteed that processing
	// loop goroutines are still running.
	select {
	case _, ok := <-c.cancel:
		if !ok {
			return errors.New("chat is already left")
		}
	default:
	}

	var message protocol.StatusMessage

	switch c.contact.Type {
	case ContactPublicChat:
		message = protocol.CreatePublicTextMessage(data, c.lastClock, c.contact.Name)
	case ContactPrivateChat:
		message = protocol.CreatePrivateTextMessage(data, c.lastClock, c.contact.Name)
	default:
		return fmt.Errorf("failed to send message: unsupported contact type")
	}

	encodedMessage, err := protocol.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to encode message")
	}

	c.Lock()
	c.updateLastClock(message.Clock)
	c.Unlock()

	opts, err := extendSendOptions(protocol.SendOptions{}, c)
	if err != nil {
		return errors.Wrap(err, "failed to prepare send options")
	}

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
	for {
		select {
		case m := <-messages:
			if c.HasMessage(m) {
				break
			}

			rearranged := c.handleMessages(m)

			if err := c.saveMessages(m); err != nil {
				c.Lock()
				c.err = err
				c.Unlock()

				close(cancel)

				return
			}

			if rearranged {
				c.onMessagesRearrange()
			} else {
				c.onNewMessage(m)
			}
		case <-sub.Done():
			c.err = sub.Err()
			close(cancel)
			return
		case <-cancel:
			return
		}
	}
}

func (c *Chat) readOwnMessagesLoop(messages <-chan *protocol.Message, cancel chan struct{}) {
	for {
		select {
		case m := <-messages:
			if c.HasMessage(m) {
				break
			}

			rearranged := c.handleMessages(m)

			if err := c.saveMessages(m); err != nil {
				c.Lock()
				c.err = err
				c.Unlock()

				close(cancel)

				return
			}

			if rearranged {
				c.onMessagesRearrange()
			} else {
				c.onNewMessage(m)
			}
		case <-cancel:
			return
		}
	}
}

func (c *Chat) handleMessages(messages ...*protocol.Message) (rearranged bool) {
	c.Lock()
	defer c.Unlock()

	for _, message := range messages {
		c.updateLastClock(message.Decoded.Clock)

		hash := messageHashStr(message)

		// TODO: remove from here
		if _, ok := c.messagesByHash[hash]; ok {
			continue
		}

		c.messagesByHash[hash] = message
		c.messages = append(c.messages, message)

		sorted := sort.SliceIsSorted(c.messages, c.lessFn)
		log.Printf("[Chat::handleMessages] sorted = %t", sorted)
		if !sorted {
			sort.SliceStable(c.messages, c.lessFn)
			rearranged = true
		}
	}

	return
}

func (c *Chat) lessFn(i, j int) bool {
	return c.messages[i].Decoded.Clock < c.messages[j].Decoded.Clock
}

func (c *Chat) onMessagesRearrange() {
	log.Printf("[Chat::onMessagesRearrange] sending EventTypeRearrange")
	c.events <- baseEvent{contact: c.contact, typ: EventTypeRearrange}
	log.Printf("[Chat::onMessagesRearrange] sent EventTypeRearrange")
}

func (c *Chat) onNewMessage(m *protocol.Message) {
	c.events <- messageEvent{
		baseEvent: baseEvent{
			contact: c.contact,
			typ:     EventTypeMessage,
		},
		message: m,
	}
}

func (c *Chat) saveMessages(messages ...*protocol.Message) error {
	return c.db.SaveMessages(c.contact, messages)
}

func (c *Chat) updateLastClock(clock int64) {
	if clock > c.lastClock {
		c.lastClock = clock
	}
}

func messageHashStr(m *protocol.Message) string {
	return hex.EncodeToString(m.Hash)
}
