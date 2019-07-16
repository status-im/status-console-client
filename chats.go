package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
	status "github.com/status-im/status-protocol-go"
)

const (
	refreshInterval = time.Second
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

	quit chan struct{}
	once sync.Once
}

// NewChatsViewController returns a new chat view controller.
func NewChatsViewController(vm *ViewController, db *sqlitePersistence, m *status.Messenger) *ChatsViewController {
	return &ChatsViewController{
		ViewController: vm,
		db: db,
		messenger: m,
		quit: make(chan struct{}),
	}
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

// load loads chats from the storage.
func (c *ChatsViewController) load() error {
	chats, err := c.db.Chats()
	if err != nil {
		return err
	}

	c.chats = chats
	return nil
}

// LoadAndRefresh loads chats from the storage and refreshes the view.
func (c *ChatsViewController) LoadAndRefresh() error {
	c.once.Do(func() {
		go func() {
			ticker := time.Tick(refreshInterval)
			for {
				select {
				case <-ticker:
					_ = c.refreshOnChanges()
				case <-c.quit:
					return
				}
			}

		}()
	})
	if err := c.load(); err != nil {
		return err
	}
	c.refresh()
	return nil
}

func (c *ChatsViewController) refreshOnChanges() error {
	chats, err := c.db.Chats()
	if err != nil {
		return err
	}
	if c.containsChanges(chats) {
		log.Printf("[chats] new chats %v", chats)
		c.chats = chats
		c.refresh()
	}
	return nil
}

func (c *ChatsViewController) containsChanges(chats []Chat) bool {
	if len(chats) != len(c.chats) {
		return true
	}
	// every time chats are sorted in a same way.
	for i := range chats {
		if !chats[i].Equal(c.chats[i]) {
			return true
		}
	}
	return false
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
