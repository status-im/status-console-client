package main

import (
	"fmt"

	"github.com/jroimartin/gocui"
	status "github.com/status-im/status-protocol-go"
)

// chatToString returns a string representation.
func chatToString(c *status.Chat) string {
	switch c.ChatType {
	case status.ChatTypePublic:
		return fmt.Sprintf("#%s", c.Name)
	case status.ChatTypeOneToOne:
		return fmt.Sprintf("@%s", c.Name)
	default:
		return c.Name
	}
}

// ChatsViewController manages chats view.
type ChatsViewController struct {
	*ViewController
	messenger *status.Messenger
	chats     []*status.Chat
}

// NewChatsViewController returns a new chat view controller.
func NewChatsViewController(vm *ViewController, m *status.Messenger) *ChatsViewController {
	return &ChatsViewController{
		ViewController: vm,
		messenger:      m,
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
func (c *ChatsViewController) ChatByIdx(idx int) (*status.Chat, bool) {
	if idx > -1 && idx < len(c.chats) {
		return c.chats[idx], true
	}
	return nil, false
}

// Add adds a new chat to the list.
func (c *ChatsViewController) Add(chat *status.Chat) error {
	if err := c.messenger.SaveChat(*chat); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// Remove removes a chat from the list.
func (c *ChatsViewController) Remove(chat *status.Chat) error {
	if err := c.messenger.DeleteChat(chat.ID, chat.ChatType); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// load loads chats from the storage.
func (c *ChatsViewController) load() error {
	chats, err := c.messenger.Chats(0, -1)
	if err != nil {
		return err
	}
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
