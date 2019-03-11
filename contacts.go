package main

import (
	"fmt"

	"github.com/jroimartin/gocui"
	"github.com/status-im/status-console-client/protocol/client"
)

// String returns a string representation.
func ContactString(c client.Contact) string {
	switch c.Type {
	case client.ContactPublicChat:
		return fmt.Sprintf("#%s", c.Name)
	case client.ContactPrivateChat:
		return fmt.Sprintf("@%s", c.Name)
	default:
		return c.Name
	}
}

// ContactsViewController manages contacts view.
type ContactsViewController struct {
	*ViewController
	messenger *client.Messenger
	contacts  []client.Contact
}

// NewContactsViewController returns a new contact view controller.
func NewContactsViewController(vm *ViewController, m *client.Messenger) *ContactsViewController {
	return &ContactsViewController{ViewController: vm, messenger: m}
}

// ContactByIdx allows to retrieve a contact for a given index.
func (c *ContactsViewController) ContactByIdx(idx int) (client.Contact, bool) {
	if idx > -1 && idx < len(c.contacts) {
		return c.contacts[idx], true
	}
	return client.Contact{}, false
}

// Refresh repaints the current list of contacts.
func (c *ContactsViewController) Refresh() {
	c.g.Update(func(*gocui.Gui) error {
		if err := c.Clear(); err != nil {
			return err
		}

		for _, contact := range c.contacts {
			if _, err := fmt.Fprintln(c.ViewController, ContactString(contact)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *ContactsViewController) Load() error {
	contacts, err := c.messenger.Contacts()
	if err != nil {
		return err
	}

	c.contacts = contacts

	return nil
}

func (c *ContactsViewController) Add(contact client.Contact) error {
	c.contacts = append(c.contacts, contact)
	return c.messenger.AddContact(contact)
}

func (c *ContactsViewController) Remove(contact client.Contact) error {
	return c.messenger.RemoveContact(contact)
}
