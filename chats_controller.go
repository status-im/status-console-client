package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/jroimartin/gocui"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/identity/alias"
)

// chatToString returns a string representation.
func chatToString(c *protocol.Chat) string {
	switch c.ChatType {
	case protocol.ChatTypePublic:
		return fmt.Sprintf("#%s", c.Name)
	case protocol.ChatTypeOneToOne:
		alias, err := alias.GenerateFromPublicKeyString(c.ID)
		if err != nil {
			return fmt.Sprintf("@%s", c.ID)
		}

		pk, err := c.PublicKey()
		if err != nil {
			return fmt.Sprintf("@%s", c.ID)
		}
		return fmt.Sprintf("@%s %#x", alias, crypto.FromECDSAPub(pk)[:8])
	default:
		return c.Name
	}
}

// ChatsViewController manages chats view.
type ChatsViewController struct {
	*ViewController
	messenger *protocol.Messenger
	chats     []*protocol.Chat
	logger    *zap.Logger
}

// NewChatsViewController returns a new chat view controller.
func NewChatsViewController(vm *ViewController, m *protocol.Messenger, logger *zap.Logger) *ChatsViewController {
	return &ChatsViewController{
		ViewController: vm,
		messenger:      m,
		logger:         logger.With(zap.Namespace("ChatsViewController")),
	}
}

// LoadAndRefresh loads chats from the storage and refreshes the view.
func (c *ChatsViewController) LoadAndRefresh() error {
	if err := c.load(); err != nil {
		return err
	}
	c.refresh()
	return nil
}

// ChatByIdx allows to retrieve a chat for a given index.
func (c *ChatsViewController) ChatByIdx(idx int) (*protocol.Chat, bool) {
	if idx > -1 && idx < len(c.chats) {
		return c.chats[idx], true
	}
	return nil, false
}

// Add adds a new chat to the list.
func (c *ChatsViewController) Add(chat protocol.Chat) error {
	if err := c.messenger.SaveChat(&chat); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// Remove removes a chat from the list.
func (c *ChatsViewController) Remove(chat protocol.Chat) error {
	if err := c.messenger.DeleteChat(chat.ID); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// load loads chats from the storage.
func (c *ChatsViewController) load() error {
	chats := c.messenger.Chats()
	c.logger.Info("loaded chats", zap.Int("count", len(chats)))
	c.chats = chats
	return nil
}

// refresh repaints the current list of chats.
func (c *ChatsViewController) refresh() {
	c.g.Update(func(*gocui.Gui) error {
		if err := c.Clear(); err != nil {
			return err
		}
		for _, chat := range c.chats {
			if _, err := fmt.Fprintln(c.ViewController, chatToString(chat)); err != nil {
				return err
			}
		}
		return nil
	})
}
