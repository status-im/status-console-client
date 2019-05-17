package main

import (
	"context"
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

	contact client.Contact

	identity  *ecdsa.PrivateKey
	messenger *client.MessengerV2

	onError func(error)

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish

	changeContact chan client.Contact
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, m *client.MessengerV2, onError func(error)) *ChatViewController {
	if onError == nil {
		onError = func(error) {}
	}

	return &ChatViewController{
		ViewController: vc,
		identity:       id,
		messenger:      m,
		onError:        onError,
		changeContact:  make(chan client.Contact, 1),
	}
}

func (c *ChatViewController) readEventsLoop(contact client.Contact) {
	c.done = make(chan struct{})
	defer close(c.done)

	var (
		messages = []*protocol.Message{}
		clock    int64
		inorder  bool
	)

	// We use a ticker in order to buffer storm of received events.
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if !inorder {
				// messages are sorted by clock value
				// TODO draw messages only after offset (if possible)
				all, err := c.messenger.Messages(c.contact, 0)
				if err != nil {
					c.onError(err)
					continue
				}
				if len(all) != 0 {
					clock = all[len(all)-1].Clock
				}
				log.Printf("[ChatViewController::readEventsLoops] retrieved %d messages", len(messages))
				c.printMessages(true, all...)
				inorder = true
			} else {
				if len(messages) != 0 {
					c.printMessages(false, messages...)
				}
			}
			messages = []*protocol.Message{}
		case event := <-c.messenger.Events():
			log.Printf("[ChatViewController::readEventsLoops] received an event: %+v", event)

			switch ev := event.(type) {
			case client.EventWithError:
				c.onError(ev.GetError())
			case client.EventWithContact:
				log.Printf("[ChatViewController::readEventsLoops] selected contact %v, msg contact %v\n", contact, ev.GetContact())
				if !ev.GetContact().Equal(contact) {
					continue
				}
				msgev, ok := ev.(client.EventWithMessage)
				if !ok {
					continue
				}
				if !inorder {
					continue
				}
				msg := msgev.GetMessage()
				log.Printf("[ChatViewController::readEventsLoops] received message current clock %v - msg clock %v\n", clock, msg.Clock)
				if msg.Clock < clock {
					inorder = false
					continue
				}
				messages = append(messages, msg)
			}
		case contact = <-c.changeContact:
			inorder = false
			clock = 0
			messages = []*protocol.Message{}
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
		go c.readEventsLoop(contact)
	}
	c.changeContact <- contact
	c.contact = contact

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return c.messenger.Join(ctx, contact)
}

// RequestOptions returns the RequestOptions for the next request call.
// Newest param when true means that we are interested in the most recent messages.
func (c *ChatViewController) RequestOptions(newest bool) (protocol.RequestOptions, error) {
	return protocol.DefaultRequestOptions(), nil
}

// RequestMessages sends a request fro historical messages.
func (c *ChatViewController) RequestMessages(params protocol.RequestOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return c.messenger.Request(ctx, c.contact, params)
}

// Send sends a payload as a message.
func (c *ChatViewController) Send(data []byte) error {
	return c.messenger.Send(c.contact, data)
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
		message.ID,
		int64(message.Clock),
		message.Timestamp.Time(),
		message.Text,
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

func formatMessageLine(id *ecdsa.PublicKey, hash []byte, clock int64, t time.Time, text string) string {
	author := "<unknown>"
	if id != nil {
		author = "0x" + hex.EncodeToString(crypto.CompressPubkey(id))[:7]
	}
	return fmt.Sprintf(
		"%s | %#+x | %d | %s | %s",
		author,
		hash[:3],
		clock,
		t.Format(time.RFC822),
		strings.TrimSpace(text),
	)
}
