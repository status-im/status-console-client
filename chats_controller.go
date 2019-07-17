package main

import (
	"fmt"

	"github.com/jroimartin/gocui"
	status "github.com/status-im/status-protocol-go"
)

// chatToString returns a string representation.
func chatToString(c Chat) string {
	switch c.Type {
	case PublicChat:
		return fmt.Sprintf("#%s", c.Name)
	case OneToOneChat:
		return fmt.Sprintf("@%s", c.Name)
	default:
		return c.Name
	}
}

// ChatsViewController manages chats view.
type ChatsViewController struct {
	*ViewController
	db        *sqlitePersistence
	messenger *status.Messenger
	chats     []Chat
}

// NewChatsViewController returns a new chat view controller.
func NewChatsViewController(vm *ViewController, db *sqlitePersistence, m *status.Messenger) *ChatsViewController {
	return &ChatsViewController{
		ViewController: vm,
		db:             db,
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
func (c *ChatsViewController) ChatByIdx(idx int) (Chat, bool) {
	if idx > -1 && idx < len(c.chats) {
		return c.chats[idx], true
	}
	return Chat{}, false
}

// Add adds a new chat to the list.
func (c *ChatsViewController) Add(chat Chat) error {
	if err := c.db.AddChats(chat); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// Remove removes a chat from the list.
func (c *ChatsViewController) Remove(chat Chat) error {
	if err := c.db.DeleteChat(chat); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// load loads chats from the storage.
func (c *ChatsViewController) load() error {
	chats, err := c.db.Chats()
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
