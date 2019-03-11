package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"

	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
)

// ChatViewController manages chat view.
type ChatViewController struct {
	*ViewController

	notifications *Notifications

	contact client.Contact

	identity  *ecdsa.PrivateKey
	messenger *client.Messenger

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, m *client.Messenger) (*ChatViewController, error) {
	return &ChatViewController{
		ViewController: vc,
		notifications:  &Notifications{writer: vc},
		identity:       id,
		messenger:      m,
	}, nil
}

func (c *ChatViewController) readEventsLoop() {
	c.done = make(chan struct{})
	defer close(c.done)

	for {
		select {
		case event := <-c.messenger.Events():
			log.Printf("received an event: %+v", event)

			switch ev := event.(type) {
			case client.EventError:
				c.notifications.Error("error", ev.Error().Error()) // nolint: errcheck
			case client.Event:
				messages, err := c.messenger.Messages(c.contact)
				if err != nil {
					c.notifications.Error("getting messages", err.Error()) // nolint: errcheck
					break
				}
				c.printMessages(messages)
			}
		case <-c.cancel:
			return
		}
	}
}

// Select informs the chat view controller about a selected contact.
// The chat view controller setup subscribers and request recent messages.
func (c *ChatViewController) Select(contact client.Contact) error {
	log.Printf("selected contact %s", contact.Name)

	if c.cancel == nil {
		c.cancel = make(chan struct{})
		go c.readEventsLoop()
	}

	c.contact = contact

	return c.messenger.Join(contact)
}

func (c *ChatViewController) RequestMessages(params protocol.RequestOptions) error {
	c.notifications.Debug( // nolint: errcheck
		"REQUEST",
		fmt.Sprintf("get historic messages: %+v", params),
	)
	return c.messenger.Request(c.contact, params)
}

func (c *ChatViewController) Send(data []byte) error {
	return c.messenger.Send(c.contact, data)
}

func (c *ChatViewController) printMessages(messages []*protocol.Message) {
	c.g.Update(func(*gocui.Gui) error {
		if err := c.Clear(); err != nil {
			return err
		}

		for _, message := range messages {
			if err := c.printMessage(message); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *ChatViewController) printMessage(message *protocol.Message) error {
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
