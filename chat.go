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

	firstRequest protocol.RequestOptions
	lastRequest  protocol.RequestOptions

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
		log.Printf("[ChatViewController::readEventsLoops] waiting for events")

		select {
		case event := <-c.messenger.Events():
			log.Printf("[ChatViewController::readEventsLoops] received an event: %+v", event)

			switch ev := event.(type) {
			case client.EventError:
				c.notifications.Error("error", ev.Error().Error()) // nolint: errcheck
			case client.EventMessage:
				c.printMessages(false, ev.Message())
			case client.Event:
				if ev.Type() != client.EventTypeInit && ev.Type() != client.EventTypeRearrange {
					break
				}

				chat := c.messenger.Chat(c.contact)
				if chat == nil {
					c.notifications.Error("getting chat", "chat does not exist") // nolint: errcheck
					break
				}

				messages := chat.Messages()

				log.Printf("[ChatViewController::readEventsLoops] retrieved %d messages", len(messages))

				c.printMessages(true, messages...)
			}
		case <-c.cancel:
			return
		}
	}
}

// Select informs the chat view controller about a selected contact.
// The chat view controller setup subscribers and request recent messages.
func (c *ChatViewController) Select(contact client.Contact) error {
	log.Printf("[ChatViewController::Select] contact %s", contact.Name)

	if c.cancel == nil {
		c.cancel = make(chan struct{})
		go c.readEventsLoop()
	}

	c.contact = contact

	params := protocol.DefaultRequestOptions()
	err := c.messenger.Join(contact, params)
	if err == nil {
		c.updateRequests(params)
	}
	return err
}

func (c *ChatViewController) RequestOptions(older bool) protocol.RequestOptions {
	params := protocol.DefaultRequestOptions()

	if older && c.firstRequest != (protocol.RequestOptions{}) {
		params.From = c.firstRequest.From - 60*60*24
		params.To = c.firstRequest.From
	} else if c.lastRequest != (protocol.RequestOptions{}) {
		params.From = c.lastRequest.To
	}

	return params
}

func (c *ChatViewController) RequestMessages(params protocol.RequestOptions) error {
	c.notifications.Debug( // nolint: errcheck
		"REQUEST",
		fmt.Sprintf("get historic messages: %+v", params),
	)

	chat := c.messenger.Chat(c.contact)
	if chat == nil {
		return fmt.Errorf("chat not found")
	}

	err := chat.Request(params)
	if err == nil {
		c.updateRequests(params)
	}
	return err
}

func (c *ChatViewController) updateRequests(params protocol.RequestOptions) {
	if c.firstRequest.From == 0 || c.firstRequest.From > params.From {
		c.firstRequest = params
	}
	if c.lastRequest.To < params.To {
		c.lastRequest = params
	}
}

func (c *ChatViewController) Send(data []byte) error {
	chat := c.messenger.Chat(c.contact)
	if chat == nil {
		return fmt.Errorf("chat not found")
	}
	return chat.Send(data)
}

func (c *ChatViewController) printMessages(clear bool, messages ...*protocol.Message) {
	log.Printf("[ChatViewController::printMessages] printing %d messages", len(messages))

	c.g.Update(func(*gocui.Gui) error {
		if clear {
			if err := c.Clear(); err != nil {
				return err
			}
		}

		for _, message := range messages {
			if err := c.writeMessage(message); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *ChatViewController) writeMessage(message *protocol.Message) error {
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
