package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-term-client/protocol/v1"
)

// ReadMessagesTimeout is a timeout for checking
// if there are any new messages available.
const ReadMessagesTimeout = time.Second

var (
	// ErrUnsupportedContactType is returned when a given contact type
	// is not supported yet.
	ErrUnsupportedContactType = errors.New("unsupported contact type")
)

// ReceivedMessage contains a raw Whisper message and decoded payload.
type ReceivedMessage struct {
	Decoded  protocol.StatusMessage
	Received *whisper.ReceivedMessage
}

// RequestMessagesParams is a list of params sent while requesting historic messages.
type RequestMessagesParams struct {
	Limit int
	From  int64
	To    int64
}

// MessagesSubscription is a subscription that retrieves messages.
type MessagesSubscription interface {
	Messages() ([]*ReceivedMessage, error)
	Unsubscribe() error
}

// PublicChat provides an interface to interact with public chats.
type PublicChat interface {
	SubscribePublicChat(name string) (MessagesSubscription, error)
	SendPublicMessage(chatName string, data []byte, identity Identity) (string, error)
	RequestPublicMessages(chatName string, params RequestMessagesParams) error
}

// Chat provides an interface to interact with any chat.
type Chat interface {
	PublicChat
}

// ChatViewController manages chat view.
type ChatViewController struct {
	*ViewController

	identity Identity
	node     Chat

	currentContact Contact
	lastClockValue int64
	sentMessages   map[string]struct{}

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, node Chat) (*ChatViewController, error) {
	return &ChatViewController{
		ViewController: vc,
		identity:       id,
		node:           node,
		sentMessages:   make(map[string]struct{}),
	}, nil
}

// Select informs the chat view controller about a selected contact.
// The chat view controller setup subscribers and request recent messages.
func (c *ChatViewController) Select(contact Contact) error {
	log.Printf("selected contact %s", contact.Name)

	c.currentContact = contact

	var (
		sub MessagesSubscription
		err error
	)

	switch contact.Type {
	case ContactPublicChat:
		sub, err = c.node.SubscribePublicChat(contact.Name)
	default:
		err = ErrUnsupportedContactType
	}
	if err != nil {
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

	go c.readMessagesLoop(sub, c.cancel, c.done)

	// Request some previous messages from the current chat
	// to provide some context for the user.
	// TODO: handle pagination
	// TODO: RequestPublicMessages should return only after receiving a response.
	params := RequestMessagesParams{
		Limit: 100,
	}
	if err := c.node.RequestPublicMessages(c.currentContact.Name, params); err != nil {
		return fmt.Errorf("failed to request messages: %v", err)
	}
	return nil
}

// TODO: change done channel to err channel. Err channel should be handled by a goroutine.
func (c *ChatViewController) readMessagesLoop(sub MessagesSubscription, cancel <-chan struct{}, done chan struct{}) {
	// TODO: check calls order. `close(done)` should be the last one.
	defer func() { _ = sub.Unsubscribe() }()
	defer close(done)

	t := time.NewTimer(ReadMessagesTimeout)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			c.g.Update(func(*gocui.Gui) error {
				messages, err := sub.Messages()
				if err != nil {
					return fmt.Errorf("failed to get messages: %v", err)
				}
				log.Printf("received %d messages", len(messages))

				c.updateLastClockValue(messages)

				return c.printMessages(messages)
			})

			t.Reset(ReadMessagesTimeout)
		case <-cancel:
			return
		}
	}
}

func (c *ChatViewController) printMessages(messages []*ReceivedMessage) error {
	myPubKey := c.identity.PublicKey

	for _, message := range messages {
		pubKey := message.Received.SigToPubKey()
		line := formatMessageLine(
			pubKey,
			time.Unix(message.Decoded.Timestamp/1000, 0),
			message.Decoded.Text,
		)

		println := fmt.Fprintln
		if pubKey.X.Cmp(myPubKey.X) == 0 && pubKey.Y.Cmp(myPubKey.Y) == 0 {
			println = color.New(color.FgGreen).Fprintln
		}

		if _, err := println(c.ViewController, line); err != nil {
			return err
		}
	}

	return nil
}

func formatMessageLine(id *ecdsa.PublicKey, t time.Time, text string) string {
	author := "<unknown>"
	if id != nil {
		author = "0x" + hex.EncodeToString(crypto.CompressPubkey(id))
	}
	return fmt.Sprintf(
		"%s | %s | %s",
		author,
		t.Format(time.RFC822),
		strings.TrimSpace(text),
	)
}

func (c *ChatViewController) updateLastClockValue(messages []*ReceivedMessage) {
	size := len(messages)
	if size == 0 {
		return
	}

	m := messages[size-1]

	if m.Decoded.Clock > c.lastClockValue {
		c.lastClockValue = m.Decoded.Clock
	}
}

// SendMessage sends a message to the selected chat (contact).
// It returns a message hash and error, if the operation fails.
func (c *ChatViewController) SendMessage(content []byte) (string, error) {
	text := string(content)
	ts := time.Now().Unix() * 1000
	clock := protocol.CalcMessageClock(c.lastClockValue, ts)
	// TODO: protocol package should expose a function to create
	// a standard StatusMessage.
	sm := protocol.StatusMessage{
		Text:      text,
		ContentT:  protocol.ContentTypeTextPlain,
		MessageT:  protocol.MessageTypePublicGroupUserMessage,
		Clock:     clock,
		Timestamp: ts,
		Content:   protocol.StatusMessageContent{ChatID: c.currentContact.Name, Text: text},
	}
	data, err := protocol.EncodeMessage(sm)
	if err != nil {
		return "", err
	}
	log.Printf("sending a message: %s", data)

	c.lastClockValue = clock

	switch c.currentContact.Type {
	case ContactPublicChat:
		return c.node.SendPublicMessage(c.currentContact.Name, data, c.identity)
	default:
		return "", ErrUnsupportedContactType
	}
}
