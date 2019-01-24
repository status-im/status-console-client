package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jroimartin/gocui"
)

// Types of contacts.
const (
	ContactPublicChat int = iota + 1
	ContactPrivateChat
)

// NewContactWithPublicKey creates a new private contact.
func NewContactWithPublicKey(name, pubKeyHex string) (c Contact, err error) {
	c.Name = name
	c.Type = ContactPrivateChat

	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return
	}

	c.publicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)
	return
}

// Contact is a single contact which has a type and name.
type Contact struct {
	Name string
	Type int

	publicKey *ecdsa.PublicKey
}

// String returns a string representation.
func (c Contact) String() string {
	switch c.Type {
	case ContactPublicChat:
		return fmt.Sprintf("#%s", c.Name)
	case ContactPrivateChat:
		return fmt.Sprintf("@%s", c.Name)
	default:
		return c.Name
	}
}

// PubKey returns a public key assigned to a contact.
// It is not nil only for private chats.
func (c Contact) PubKey() *ecdsa.PublicKey {
	return c.publicKey
}

// ContactsViewController manages contacts view.
type ContactsViewController struct {
	*ViewController
	items []Contact
}

// NewContactsViewController returns a new contact view controller.
func NewContactsViewController(vm *ViewController, items []Contact) *ContactsViewController {
	return &ContactsViewController{vm, items}
}

// ContactByIdx allows to retrieve a contact for a given index.
func (c *ContactsViewController) ContactByIdx(idx int) (Contact, bool) {
	if idx > -1 && idx < len(c.items) {
		return c.items[idx], true
	}
	return Contact{}, false
}

// Refresh repaints the current list of contacts.
func (c *ContactsViewController) Refresh() {
	c.g.Update(func(*gocui.Gui) error {
		if err := c.Clear(); err != nil {
			return err
		}

		for _, item := range c.items {
			if _, err := fmt.Fprintln(c.ViewController, item); err != nil {
				return err
			}
		}
		return nil
	})
}
