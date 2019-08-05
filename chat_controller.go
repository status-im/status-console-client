package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"

	status "github.com/status-im/status-protocol-go"
	protocol "github.com/status-im/status-protocol-go/v1"
)

// ChatViewController manages chat view.
type ChatViewController struct {
	*ViewController

	identity  *ecdsa.PrivateKey
	messenger *status.Messenger

	chat       *status.Chat
	onError    func(error)
	changeChat chan *status.Chat

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewChatViewController returns a new chat view controller.
func NewChatViewController(vc *ViewController, id Identity, m *status.Messenger, onError func(error)) *ChatViewController {
	if onError == nil {
		onError = func(error) {}
	}

	return &ChatViewController{
		ViewController: vc,
		identity:       id,
		messenger:      m,
		onError:        onError,
		changeChat:     make(chan *status.Chat, 1),
	}
}

func (c *ChatViewController) readMessagesLoop() {
	var chat *status.Chat

	c.done = make(chan struct{})
	defer close(c.done)

	// A list of all messages displayed on the screen.
	// TODO: It should be a round buffer instead.
	var messages []*protocol.Message

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if chat.ID == "" {
				continue
			}

			latest, err := c.retrieveMessagesForChat(chat, status.RetrieveLatest)
			if err != nil {
				log.Printf("[ChatViewController::readMessagesLoop] failed to retrieve messages: %v", err)
				continue
			}
			log.Printf("[ChatViewController::readMessagesLoop] retrieved %d messages", len(latest))

			latest = filterMessages(latest, &c.identity.PublicKey, chat)
			log.Printf("[ChatViewController::readMessagesLoop] after filtering %d left", len(latest))

			if len(latest) == 0 {
				break
			}

			lastClock := int64(0)
			if len(messages) > 0 {
				lastClock = messages[len(messages)-1].Clock
			}

			repaint := false
			for _, l := range latest {
				if l.Clock < lastClock {
					repaint = true
					break
				}
			}

			messages = append(messages, latest...)

			if !repaint {
				sortMessages(latest)
				c.printMessages(false, latest...)
				break
			}

			sortMessages(messages)
			c.printMessages(true, messages...)
		case chat = <-c.changeChat:
			latest, err := c.retrieveMessagesForChat(chat, status.RetrieveLastDay)
			if err != nil {
				exitErr(err)
			}
			messages = filterMessages(latest, &c.identity.PublicKey, chat)
			c.printMessages(true, messages...)
		case <-c.cancel:
			return
		}
	}
}

func sortMessages(messages []*protocol.Message) {
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Clock < messages[j].Clock
	})
}

func filterMessages(messages []*protocol.Message, myPublicKey *ecdsa.PublicKey, c *status.Chat) (result []*protocol.Message) {
	publicKeyBytes := crypto.FromECDSAPub(c.PublicKey)
	myPublicKeyBytes := crypto.FromECDSAPub(myPublicKey)
	for _, m := range messages {
		if c.PublicKey != nil {
			sigBytes := crypto.FromECDSAPub(m.SigPubKey)
			if bytes.Equal(myPublicKeyBytes, sigBytes) || bytes.Equal(publicKeyBytes, sigBytes) {
				result = append(result, m)
			} else {
				log.Printf("[filterMessages] expected public key %x got %x", publicKeyBytes, sigBytes)
			}
		} else {
			if c.ID == m.Content.ChatID {
				result = append(result, m)
			} else {
				log.Printf("[filterMessages] expected chatID %s got %s", c.ID, m.Content.ChatID)
			}
		}
	}
	return
}

func (c *ChatViewController) retrieveMessagesForChat(chat *status.Chat, rConfig status.RetrieveConfig) ([]*protocol.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c.messenger.Retrieve(ctx, *chat, rConfig)
}

// Select informs the chat view controller about a selected contact.
// The chat view controller setup subscribers and request recent messages.
func (c *ChatViewController) Select(chat *status.Chat) {
	log.Printf("[ChatViewController::Select] chat %s", chat.ID)

	if c.cancel == nil {
		c.cancel = make(chan struct{})
		go c.readMessagesLoop()
	}
	c.changeChat <- chat
	c.chat = chat
}

// Send sends a payload as a message.
func (c *ChatViewController) Send(ctx context.Context, data []byte) ([]byte, error) {
	log.Printf("[ChatViewController::Send] chat %s", c.chat.ID)
	return c.messenger.Send(ctx, *c.chat, data)
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
