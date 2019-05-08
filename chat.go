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
	messenger *client.Messenger

	onError func(error)

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, m *client.Messenger, onError func(error)) *ChatViewController {
	if onError == nil {
		onError = func(error) {}
	}

	return &ChatViewController{
		ViewController: vc,
		identity:       id,
		messenger:      m,
		onError:        onError,
	}
}

func (c *ChatViewController) readEventsLoop() {
	c.done = make(chan struct{})
	defer close(c.done)

	// buckets with events indexed by contacts
	buffer := make(map[client.Contact][]interface{})

	// We use a ticker in order to buffer storm of received events.
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			chat := c.messenger.Chat(c.contact)
			if chat == nil {
				c.onError(fmt.Errorf("no chat for contact '%s'", c.contact))
				break
			}

			needsRedraw := requiresRedraw(buffer[c.contact])

			if needsRedraw {
				// Get all available messages and
				// rewrite the buffer completely.
				messages := chat.Messages()

				log.Printf("[ChatViewController::readEventsLoops] retrieved %d messages", len(messages))

				c.printMessages(true, messages...)
			} else {
				// Messages arrived in order so we can safely put them
				// at the end of the buffer.
				for _, event := range buffer[c.contact] {
					switch ev := event.(type) {
					case client.EventWithMessage:
						c.printMessages(false, ev.GetMessage())
					}
				}
			}

			buffer = make(map[client.Contact][]interface{})
		case event := <-c.messenger.Events():
			log.Printf("[ChatViewController::readEventsLoops] received an event: %+v", event)

			switch ev := event.(type) {
			case client.EventWithError:
				c.onError(ev.GetError())
			case client.EventWithContact:
				buffer[ev.GetContact()] = append(buffer[ev.GetContact()], ev)
			}
		case <-c.cancel:
			return
		}
	}
}

// requiresRedraw checks if in the list of events, there are any which
// require to redraw the whole view. This is an optimization as usually
// messages arrive in order so they can simply be appended to the buffer,
// however, if the message is our of order the whole buffer needs to be changed.
func requiresRedraw(events []interface{}) bool {
	for _, event := range events {
		switch ev := event.(type) {
		case client.EventWithType:
			switch ev.GetType() {
			case client.EventTypeInit, client.EventTypeRearrange:
				return true
			}
		}
	}
	return false
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_, err := c.messenger.Join(ctx, contact)
	return err
}

// RequestOptions returns the RequestOptions for the next request call.
// Newest param when true means that we are interested in the most recent messages.
func (c *ChatViewController) RequestOptions(newest bool) (protocol.RequestOptions, error) {
	chat := c.messenger.Chat(c.contact)
	if chat == nil {
		return protocol.RequestOptions{}, fmt.Errorf("chat not found")
	}
	return chat.RequestOptions(newest)
}

// RequestMessages sends a request fro historical messages.
func (c *ChatViewController) RequestMessages(params protocol.RequestOptions) error {
	chat := c.messenger.Chat(c.contact)
	if chat == nil {
		return fmt.Errorf("chat not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return chat.Request(ctx, params)
}

// Send sends a payload as a message.
func (c *ChatViewController) Send(data []byte) error {
	chat := c.messenger.Chat(c.contact)
	if chat == nil {
		return fmt.Errorf("chat not found")
	}
	return chat.SendMessage(data)
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
