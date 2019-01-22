package main

import (
	"context"
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

// ChatViewController manages chat view.
type ChatViewController struct {
	*ViewController

	identity *ecdsa.PrivateKey
	chat     protocol.Chat

	currentContact Contact
	lastClockValue int64
	sentMessages   map[string]struct{}

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, chat protocol.Chat) (*ChatViewController, error) {
	return &ChatViewController{
		ViewController: vc,
		identity:       id,
		chat:           chat,
		sentMessages:   make(map[string]struct{}),
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
		if err != nil {
			err = fmt.Errorf("failed to subscribe to public chat: %v", err)
		}
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

	go c.readMessagesLoop(sub, messages, c.cancel, c.done)

	// Request some previous messages from the current chat
	// to provide some context for the user.
	// TODO: handle pagination
	// TODO: RequestPublicMessages should return only after receiving a response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	params := protocol.RequestMessagesParams{
		Limit: 100,
	}
	if err := c.chat.RequestPublicMessages(ctx, c.currentContact.Name, params); err != nil {
		return fmt.Errorf("failed to request messages: %v", err)
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

	for {
		select {
		case m := <-messages:
			c.updateLastClockValue(m)

			c.g.Update(func(*gocui.Gui) error {
				err := c.printMessage(m)
				if err != nil {
					err = fmt.Errorf("failed to print a message because of %v", err)
				}
				return err
			})
		case <-sub.Done():
			if err := sub.Err(); err != nil {
				log.Fatalf("protocol subscription errored: %v", err)
			}
		case <-cancel:
			return
		}
	}
}

func (c *ChatViewController) printMessage(message *protocol.ReceivedMessage) error {
	myPubKey := c.identity.PublicKey
	pubKey := message.SigPubKey

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

func (c *ChatViewController) updateLastClockValue(m *protocol.ReceivedMessage) {
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		return c.chat.SendPublicMessage(ctx, c.currentContact.Name, data, c.identity)
	default:
		return "", ErrUnsupportedContactType
	}
}
