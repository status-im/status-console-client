package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/protobuf"
)

// MessagesViewController manages chat view.
type MessagesViewController struct {
	*ViewController

	// TODO: It should be a round buffer instead.
	// It is a map with chatID as a key and a list of messages.
	store          map[string][]*protocol.Message
	mutex          sync.Mutex
	identity       *ecdsa.PrivateKey
	myPubkeyString string
	messenger      *protocol.Messenger
	logger         *zap.Logger

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
		myPubkeyString: "0x" + hex.EncodeToString(crypto.FromECDSAPub(&id.PublicKey)),
		store:          make(map[string][]*protocol.Message),
		messenger:      m,
		logger:         logger.With(zap.Namespace("MessagesViewController")),
		onMessages:     onMessages,
		onError:        onError,
		changeChat:     make(chan *protocol.Chat, 1),
	}
}

func (c *MessagesViewController) Start() error {

	if c.cancel == nil {
		c.cancel = make(chan struct{})
		chats, err := c.messenger.Chats()
		if err != nil {
			return err
		}

		for _, chat := range chats {
			// Pull latest 10 messages
			latestMessages, _, err := c.messenger.MessageByChatID(chat.ID, "", 10)
			if err != nil {
				return err
			}

			c.mutex.Lock()
			c.store[chat.ID] = append(c.store[chat.ID], latestMessages...)
			c.mutex.Unlock()
		}

		go c.readMessagesLoop()
	}
	return nil
}

func (c *MessagesViewController) handleRetrievedMessages(response *protocol.MessengerResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, m := range response.Messages {
		c.store[m.LocalChatID] = append(c.store[m.LocalChatID], m)
	}

	c.onMessages()

	if c.activeChat == nil {
		return
	}

	var latestForActive []*protocol.Message
	for _, m := range response.Messages {
		if m.LocalChatID == c.activeChat.ID {
			latestForActive = append(latestForActive, m)
		}
	}

	if len(latestForActive) == 0 {
		return
	}

	var messagesToDraw []*protocol.Message

	repaint := isRepaintNeeded(latestForActive, c.store[c.activeChat.ID])
	if repaint {
		messagesToDraw = c.store[c.activeChat.ID]
	} else {
		messagesToDraw = latestForActive
	}

	sortMessages(messagesToDraw)
	c.printMessages(repaint, messagesToDraw...)
}

func (c *MessagesViewController) readMessagesLoop() {
	c.done = make(chan struct{})
	defer close(c.done)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			response, err := c.messenger.RetrieveAll()
			if err != nil {
				c.logger.Error("failed to retrieve messages", zap.Error(err))
				continue
			}

			c.logger.Info("received latest messages", zap.Int("count", len(response.Messages)))
			c.handleRetrievedMessages(response)

		case chat := <-c.changeChat:
			c.activeChat = chat
			c.logger.Info("changed active chat", zap.Int("count", len(c.store[chat.ID])))
			c.printMessages(true, c.store[chat.ID]...)
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

func isRepaintNeeded(latest, messages []*protocol.Message) bool {
	lastClock := uint64(0)
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
func (c *MessagesViewController) Send(ctx context.Context, text string) (*protocol.MessengerResponse, error) {
	if c.activeChat == nil {
		return nil, errors.New("no selected chat")
	}
	c.logger.Info("sending message", zap.String("chatID", c.activeChat.ID), zap.String("text", text))
	message := &protocol.Message{}
	message.ChatId = c.activeChat.ID
	message.Text = text
	message.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	response, err := c.messenger.SendChatMessage(ctx, message)
	if err != nil {
		return nil, err
	}
	m := response.Messages[0]

	c.mutex.Lock()
	c.store[m.LocalChatID] = append(c.store[m.LocalChatID], m)

	c.printMessages(false, m)
	c.mutex.Unlock()

	return response, nil
}

func (c *MessagesViewController) printMessages(clear bool, messages ...*protocol.Message) {
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

func (c *MessagesViewController) writeMessage(message *protocol.Message) error {

	line := formatMessageLine(
		message.Alias,
		message.From,
		message.ID,
		int64(message.Clock),
		message.WhisperTimestamp,
		message.Text,
	)

	println := fmt.Fprintln
	// TODO: extract
	if message.From == c.myPubkeyString {
		println = color.New(color.FgGreen).Fprintln
	}

	if _, err := println(c.ViewController, line); err != nil {
		return err
	}

	return nil
}

func formatMessageLine(alias string, from string, messageID string, clock int64, t uint64, text string) string {
	return fmt.Sprintf(
		"%s | %s | %#+x | %d | %s | %s",
		alias,
		from[:9],
		messageID[2:3],
		clock,
		time.Unix(int64(t), 0).Format(time.RFC822),
		strings.TrimSpace(text),
	)
}
