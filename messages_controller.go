package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

// MessagesViewController manages chat view.
type MessagesViewController struct {
	*ViewController

	identity  *ecdsa.PrivateKey
	messenger *protocol.Messenger
	logger    *zap.Logger

	activeChat *protocol.Chat
	onError    func(error)
	onMessages func()
	changeChat chan *protocol.Chat

	cancel chan struct{} // cancel the current chat loop
	done   chan struct{} // wait for the current chat loop to finish
}

// NewMessagesViewController returns a new chat view controller.
func NewMessagesViewController(
	vc *ViewController,
	id Identity,
	m *protocol.Messenger,
	logger *zap.Logger,
	onMessages func(),
	onError func(error),
) *MessagesViewController {
	if onMessages == nil {
		onMessages = func() {}
	}
	if onError == nil {
		onError = func(error) {}
	}

	return &MessagesViewController{
		ViewController: vc,
		identity:       id,
		messenger:      m,
		logger:         logger.With(zap.Namespace("MessagesViewController")),
		onMessages:     onMessages,
		onError:        onError,
		changeChat:     make(chan *protocol.Chat, 1),
	}
}

func (c *MessagesViewController) Start() {
	if c.cancel == nil {
		c.cancel = make(chan struct{})
		go c.readMessagesLoop()
	}
}

func (c *MessagesViewController) readMessagesLoop() {
	c.done = make(chan struct{})
	defer close(c.done)

	// TODO: It should be a round buffer instead.
	// It is a map with chatID as a key and a list of messages.
	store := make(map[string][]*v1protocol.Message)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			allLatest, err := c.messenger.RetrieveAll(ctx, protocol.RetrieveLatest)
			cancel()
			if err != nil {
				c.logger.Error("failed to retrieve messages", zap.Error(err))
				continue
			}

			c.logger.Debug("received latest messages", zap.Int("count", len(allLatest)))

			for _, m := range allLatest {
				store[m.ChatID] = append(store[m.ChatID], m)
			}

			c.onMessages()

			if c.activeChat == nil {
				break
			}

			var latestForActive []*v1protocol.Message
			for _, m := range allLatest {
				if m.ChatID == c.activeChat.ID {
					latestForActive = append(latestForActive, m)
				}
			}

			if len(latestForActive) == 0 {
				break
			}

			var messagesToDraw []*v1protocol.Message

			repaint := isRepaintNeeded(latestForActive, store[c.activeChat.ID])
			if repaint {
				messagesToDraw = store[c.activeChat.ID]
			} else {
				messagesToDraw = latestForActive
			}

			sortMessages(messagesToDraw)
			c.printMessages(repaint, messagesToDraw...)
		case chat := <-c.changeChat:
			c.activeChat = chat
			c.logger.Info("changed active chat", zap.Int("count", len(store[chat.ID])))
			c.printMessages(true, store[chat.ID]...)
		case <-c.cancel:
			return
		}
	}
}

func sortMessages(messages []*v1protocol.Message) {
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Clock < messages[j].Clock
	})
}

func isRepaintNeeded(latest, messages []*v1protocol.Message) bool {
	lastClock := int64(0)
	if len(messages) > 0 {
		lastClock = messages[len(messages)-1].Clock
	}
	for _, l := range latest {
		if l.Clock < lastClock {
			return true
		}
	}
	return false
}

// ActiveChat returns the active chat, if any
func (c *MessagesViewController) ActiveChat() *protocol.Chat {
	return c.activeChat
}

// Select informs the chat view controller about a selected contact.
// The chat view controller setup subscribers and request recent messages.
func (c *MessagesViewController) Select(chat *protocol.Chat) {
	c.logger.Info("selected chat", zap.String("chatID", chat.ID))
	c.changeChat <- chat
}

// Send sends a payload as a message.
func (c *MessagesViewController) Send(ctx context.Context, data []byte) ([][]byte, error) {
	if c.activeChat == nil {
		return nil, errors.New("no selected chat")
	}
	c.logger.Info("sending message", zap.String("chatID", c.activeChat.ID), zap.ByteString("data", data))
	return c.messenger.Send(ctx, c.activeChat.ID, data)
}

func (c *MessagesViewController) printMessages(clear bool, messages ...*v1protocol.Message) {
	c.logger.Debug("printing messages", zap.Int("count", len(messages)))
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

func (c *MessagesViewController) writeMessage(message *v1protocol.Message) error {
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
