package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"

	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	// ErrUnsupportedContactType is returned when a given contact type
	// is not supported yet.
	ErrUnsupportedContactType = errors.New("unsupported contact type")
)

// ChatViewController manages chat view.
type ChatViewController struct {
	*ViewController

	notifications *Notifications

	identity *ecdsa.PrivateKey
	chat     protocol.Chat

	db             *Database
	messages       []*protocol.ReceivedMessage // ordered by Clock
	messagesByHash map[string]*protocol.ReceivedMessage

	currentContact Contact
	lastClockValue int64

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, chat protocol.Chat, db *Database) (*ChatViewController, error) {
	return &ChatViewController{
		ViewController: vc,
		notifications:  &Notifications{writer: vc},
		identity:       id,
		chat:           chat,
		db:             db,
	}, nil
}

// Select informs the chat view controller about a selected contact.
// The chat view controller setup subscribers and request recent messages.
func (c *ChatViewController) Select(contact Contact) error {
	log.Printf("selected contact %s", contact.Name)

	c.currentContact = contact

	var (
		sub *protocol.Subscription
		err error
	)

	messages := make(chan *protocol.ReceivedMessage)

	switch contact.Type {
	case ContactPublicChat:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		sub, err = c.chat.SubscribePublicChat(ctx, contact.Name, messages)
	case ContactPrivateChat:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		sub, err = c.chat.SubscribePrivateChat(ctx, c.identity, messages)
	default:
		err = ErrUnsupportedContactType
	}
	if err != nil {
		err = fmt.Errorf("failed to subscribe to chat: %v", err)
		return err
	}

	// clear messages from the previous chat
	if err := c.Clear(); err != nil {
		return err
	}

	// cancel the previous loop, if exists
	if c.cancel != nil {
		close(c.cancel)
	}
	// wait for the loop to finish
	if c.done != nil {
		<-c.done
	}

	c.cancel = make(chan struct{})
	c.done = make(chan struct{})

	go c.readMessagesLoop(sub, messages, c.cancel, c.done)

	result, err := c.db.Messages(
		c.currentContact,
		DefaultRequestMessagesParams().From,
		DefaultRequestMessagesParams().To,
	)
	if err != nil {
		return fmt.Errorf("failed to get messages from db: %v", err)
	}

	log.Printf("got %d messages from the local db", len(result))

	for _, m := range result {
		messages <- m
	}

	return c.RequestMessagesWithDefaults()
}

func DefaultRequestMessagesParams() protocol.RequestMessagesParams {
	return protocol.RequestMessagesParams{
		From:  time.Now().Add(-24 * time.Hour).Unix(),
		To:    time.Now().Unix(),
		Limit: 1000,
	}
}

func (c *ChatViewController) RequestMessagesWithDefaults() error {
	return c.RequestMessages(DefaultRequestMessagesParams())
}

func (c *ChatViewController) RequestMessages(params protocol.RequestMessagesParams) error {
	c.notifications.Debug("REQUEST", fmt.Sprintf("get historic messages: %+v", params))

	// Request some previous messages from the current chat
	// to provide some context for the user.
	// TODO: handle pagination
	// TODO: RequestPublicMessages should return only after receiving a response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	switch c.currentContact.Type {
	case ContactPublicChat:
		if err := c.chat.RequestPublicMessages(ctx, c.currentContact.Name, params); err != nil {
			return fmt.Errorf("failed to request public messages: %v", err)
		}
	case ContactPrivateChat:
		if err := c.chat.RequestPrivateMessages(ctx, params); err != nil {
			return fmt.Errorf("failed to request private messages: %v", err)
		}
	default:
		return errors.New("invalid contact type")
	}

	return nil
}

// TODO: change done channel to err channel. Err channel should be handled by a goroutine.
func (c *ChatViewController) readMessagesLoop(
	sub *protocol.Subscription,
	messages <-chan *protocol.ReceivedMessage,
	cancel <-chan struct{},
	done chan struct{},
) {
	// TODO: check calls order. `close(done)` should be the last one.
	defer sub.Unsubscribe()
	defer close(done)

	c.messages = nil
	c.messagesByHash = make(map[string]*protocol.ReceivedMessage)

	for {
		select {
		case m := <-messages:
			if err := c.handleIncomingMessage(m); err != nil {
				fmt.Printf("failed to handle incoming message: %v", err)
			}
		case <-sub.Done():
			if err := sub.Err(); err != nil {
				log.Fatalf("protocol subscription errored: %v", err)
			}
		case <-cancel:
			return
		}
	}
}

func (c *ChatViewController) handleIncomingMessage(m *protocol.ReceivedMessage) error {
	lessFn := func(i, j int) bool {
		return c.messages[i].Decoded.Clock < c.messages[j].Decoded.Clock
	}
	hash := hex.EncodeToString(m.Hash)

	// the message already exists
	if _, ok := c.messagesByHash[hash]; ok {
		return nil
	}

	c.updateLastClockValue(m)

	c.messagesByHash[hash] = m
	c.messages = append(c.messages, m)

	isSorted := sort.SliceIsSorted(c.messages, lessFn)
	if !isSorted {
		sort.Slice(c.messages, lessFn)
	}

	if err := c.db.SaveMessages(c.currentContact, m); err != nil {
		return err
	}

	c.g.Update(func(*gocui.Gui) error {
		var err error

		if isSorted {
			err = c.printMessage(m)
		} else {
			err = c.reprintMessages()
		}

		if err != nil {
			err = fmt.Errorf("failed reprint messages: %v", err)
		}
		return err
	})

	return nil
}

func (c *ChatViewController) reprintMessages() error {
	if err := c.Clear(); err != nil {
		return err
	}

	for _, m := range c.messages {
		if err := c.printMessage(m); err != nil {
			return err
		}
	}

	return nil
}

func (c *ChatViewController) printMessage(message *protocol.ReceivedMessage) error {
	myPubKey := c.identity.PublicKey
	pubKey := message.SigPubKey

	line := formatMessageLine(
		pubKey,
		message.Hash,
		time.Unix(message.Decoded.Timestamp/1000, 0),
		message.Decoded.Text,
	)

	println := fmt.Fprintln
	// TODO: extract
	if pubKey.X.Cmp(myPubKey.X) == 0 && pubKey.Y.Cmp(myPubKey.Y) == 0 {
		println = color.New(color.FgGreen).Fprintln
	}

	if _, err := println(c.ViewController, line); err != nil {
		return err
	}

	return nil
}

func formatMessageLine(id *ecdsa.PublicKey, hash []byte, t time.Time, text string) string {
	author := "<unknown>"
	if id != nil {
		author = "0x" + hex.EncodeToString(crypto.CompressPubkey(id))[:7]
	}
	return fmt.Sprintf(
		"%s | %#+x | %s | %s",
		author,
		hash[:3],
		t.Format(time.RFC822),
		strings.TrimSpace(text),
	)
}

func (c *ChatViewController) updateLastClockValue(m *protocol.ReceivedMessage) {
	if m.Decoded.Clock > c.lastClockValue {
		c.lastClockValue = m.Decoded.Clock
	}
}

// SendMessage sends a message to the selected chat (contact).
// It returns a message hash and error, if the operation fails.
func (c *ChatViewController) SendMessage(content []byte) ([]byte, error) {
	text := strings.TrimSpace(string(content))
	ts := time.Now().Unix() * 1000
	clock := protocol.CalcMessageClock(c.lastClockValue, ts)

	var messageType string

	switch c.currentContact.Type {
	case ContactPublicChat:
		messageType = protocol.MessageTypePublicGroupUserMessage
	case ContactPrivateChat:
		messageType = protocol.MessageTypePrivateUserMessage
	default:
		return nil, ErrUnsupportedContactType
	}

	// TODO: protocol package should expose a function to create
	// a standard StatusMessage.
	sm := protocol.StatusMessage{
		Text:      text,
		ContentT:  protocol.ContentTypeTextPlain,
		MessageT:  messageType,
		Clock:     clock,
		Timestamp: ts,
		Content:   protocol.StatusMessageContent{ChatID: c.currentContact.Name, Text: text},
	}
	data, err := protocol.EncodeMessage(sm)
	if err != nil {
		return nil, err
	}
	log.Printf("sending a message: %s", data)

	c.lastClockValue = clock

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	switch c.currentContact.Type {
	case ContactPublicChat:
		return c.chat.SendPublicMessage(ctx, c.currentContact.Name, data, c.identity)
	case ContactPrivateChat:
		log.Printf("sending a private message: %x", crypto.FromECDSAPub(c.currentContact.PublicKey))

		hash, err := c.chat.SendPrivateMessage(ctx, c.currentContact.PublicKey, data, c.identity)
		if err != nil {
			return nil, err
		}

		m := protocol.ReceivedMessage{
			Decoded:   sm,
			SigPubKey: &c.identity.PublicKey,
			Hash:      hash,
		}

		return hash, c.handleIncomingMessage(&m)
	default:
		return nil, ErrUnsupportedContactType
	}
}
