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

	contact client.Contact

	firstRequest protocol.RequestOptions
	lastRequest  protocol.RequestOptions

	identity  *ecdsa.PrivateKey
	messenger *client.Messenger

	onError func(error)

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, m *client.Messenger, onError func(error)) *ChatViewController {
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

	var buffer []interface{}

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

			redraw := requiresRedraw(buffer)

			if redraw {
				messages := chat.Messages()

				log.Printf("[ChatViewController::readEventsLoops] retrieved %d messages", len(messages))

				c.printMessages(true, messages...)
			} else {
				for _, event := range buffer {
					switch ev := event.(type) {
					case client.EventMessage:
						c.printMessages(false, ev.Message())
					}
				}
			}

			buffer = nil
		case event := <-c.messenger.Events():
			log.Printf("[ChatViewController::readEventsLoops] received an event: %+v", event)

			switch ev := event.(type) {
			case client.EventError:
				c.onError(ev.Error())
			default:
				buffer = append(buffer, event)
			}
		case <-c.cancel:
			return
		}
	}
}

func requiresRedraw(events []interface{}) bool {
	for _, event := range events {
		switch ev := event.(type) {
		case client.Event:
			switch ev.Type() {
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
		message.Decoded.Clock,
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
