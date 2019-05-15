package main

import (
	"fmt"

	"github.com/jroimartin/gocui"
	"github.com/status-im/status-console-client/protocol/client"
)

// contactToString returns a string representation.
func contactToString(c client.Contact) string {
	switch c.Type {
	case client.ContactPublicRoom:
		return fmt.Sprintf("#%s", c.Name)
	case client.ContactPublicKey:
		return fmt.Sprintf("@%s", c.Name)
	default:
		return c.Name
	}
}

// ContactsViewController manages contacts view.
type ContactsViewController struct {
	*ViewController
	messenger *client.MessengerV2
	contacts  []client.Contact
}

// NewContactsViewController returns a new contact view controller.
func NewContactsViewController(vm *ViewController, m *client.MessengerV2) *ContactsViewController {
	return &ContactsViewController{ViewController: vm, messenger: m}
}

// refresh repaints the current list of contacts.
func (c *ContactsViewController) refresh() {
	c.g.Update(func(*gocui.Gui) error {
		if err := c.Clear(); err != nil {
			return err
		}

		for _, contact := range c.contacts {
			if _, err := fmt.Fprintln(c.ViewController, contactToString(contact)); err != nil {
				return err
			}
		}
		return nil
	})
}

// load loads contacts from the storage.
func (c *ContactsViewController) load() error {
	contacts, err := c.messenger.Contacts()
	if err != nil {
		return err
	}

	c.contacts = contacts

	return nil
}

// LoadAndRefresh loads contacts from the storage and refreshes the view.
func (c *ContactsViewController) LoadAndRefresh() error {
	if err := c.load(); err != nil {
		return err
	}
	c.refresh()
	return nil
}

// ContactByIdx allows to retrieve a contact for a given index.
func (c *ContactsViewController) ContactByIdx(idx int) (client.Contact, bool) {
	if idx > -1 && idx < len(c.contacts) {
		return c.contacts[idx], true
	}
	return client.Contact{}, false
}

// Add adds a new contact to the list.
func (c *ContactsViewController) Add(contact client.Contact) error {
	if err := c.messenger.AddContact(contact); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}

// Remove removes a contact from the list.
func (c *ContactsViewController) Remove(contact client.Contact) error {
	if err := c.messenger.RemoveContact(contact); err != nil {
		return err
	}
	return c.LoadAndRefresh()
}
