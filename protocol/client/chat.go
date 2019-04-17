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
// Instances should not be reused after
// leaving but instead a new instance
// should be created and replace the
// previous one.
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

// Events returns a channel with Chat events.
func (c *Chat) Events() <-chan interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.events
}

// Done informs when the Chat finished processing messages.
func (c *Chat) Done() <-chan struct{} {
	c.RLock()
	defer c.RUnlock()
	return c.cancel
}

// Err returns a cached error.
func (c *Chat) Err() error {
	c.RLock()
	defer c.RUnlock()
	return c.err
}

func (c *Chat) updateLastClock(clock int64) {
	if clock > c.lastClock {
		c.lastClock = clock
	}
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
	_, ok := c.hasMessageWithHash(m)
	c.RUnlock()
	return ok
}

func (c *Chat) hasMessageWithHash(m *protocol.Message) (string, bool) {
	hash := hex.EncodeToString(m.ID)
	_, ok := c.messagesByHash[hash]
	return hash, ok
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

	var message protocol.Message

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
	c.updateLastClock(int64(message.Clock))
	c.Unlock()

	opts, err := createSendOptions(c.contact)
	if err != nil {
		return errors.Wrap(err, "failed to prepare send options")
	}

	hash, err := c.proto.Send(context.Background(), encodedMessage, opts)

	// Own messages need to be pushed manually to the pipeline.
	if c.contact.Type == ContactPrivateChat {
		log.Printf("[Chat::Send] sent a private message")

		// TODO: this should be created by c.proto
		message.SigPubKey = &c.identity.PublicKey
		message.ID = hash
		c.ownMessages <- &message
	}

	return err
}

// Request historic messages.
func (c *Chat) Request(options protocol.RequestOptions) error {
	return c.request(options)
}

func (c *Chat) request(options protocol.RequestOptions) error {
	opts, err := createRequestOptions(c.contact)
	if err != nil {
		return err
	}
	return c.proto.Request(context.Background(), opts)
}

// subscribe reads messages from the network.
func (c *Chat) subscribe(params protocol.RequestOptions) error {
	c.RLock()
	cancel := c.cancel
	sub := c.sub
	c.RUnlock()

	if sub != nil {
		return errors.New("already subscribed")
	}

	opts, err := createSubscribeOptions(c.contact)
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

	go c.readLoop(messages, sub, cancel)

	// Request messages from the cache.
	if err := c.load(params); err != nil {
		return errors.Wrap(err, "failed to load cached messages")
	}

	// Request historic messages from the network.
	if err := c.request(params); err != nil {
		return errors.Wrap(err, "failed to request for messages")
	}

	return nil
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

// load loads messages from the database cache and then the network.
func (c *Chat) load(options protocol.RequestOptions) error {
	// Get already cached messages from the database.
	cachedMessages, err := c.db.Messages(
		c.contact,
		options.FromAsTime(),
		options.ToAsTime(),
	)
	if err != nil {
		return errors.Wrap(err, "db failed to get messages")
	}

	c.handleMessages(cachedMessages...)

	go func() {
		log.Printf("[Chat::load] sending EventTypeInit")
		c.events <- baseEvent{contact: c.contact, typ: EventTypeInit}
		log.Printf("[Chat::load] sent EventTypeInit")
	}()

	return nil
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

	for _, m := range messages {
		c.updateLastClock(int64(m.Clock))

		hash, exists := c.hasMessageWithHash(m)
		if exists {
			continue
		}

		c.messagesByHash[hash] = m
		c.messages = append(c.messages, m)

		sorted := sort.SliceIsSorted(c.messages, c.lessFn)
		log.Printf("[Chat::handleMessages] sorted = %t", sorted)
		if !sorted {
			sort.SliceStable(c.messages, c.lessFn)
			rearranged = true
		}
	}

	return
}

func (c *Chat) saveMessages(messages ...*protocol.Message) error {
	return c.db.SaveMessages(c.contact, messages)
}

func (c *Chat) lessFn(i, j int) bool {
	return c.messages[i].Clock < c.messages[j].Clock
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
