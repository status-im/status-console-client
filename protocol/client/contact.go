package client

import (
	"crypto/ecdsa"
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

type ContactType int

// Types of contacts.
const (
	ContactPublicChat ContactType = iota + 1
	ContactPrivateChat
)

// Contact is a single contact which has a type and name.
type Contact struct {
	Name      string
	Type      ContactType
	PublicKey *ecdsa.PublicKey
}

// ContactWithPublicKey creates a new private contact.
func ContactWithPublicKey(name, pubKeyHex string) (c Contact, err error) {
	c.Name = name
	c.Type = ContactPrivateChat

	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return
	}

	c.PublicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)
	return
}

func ContainsContact(cs []Contact, c Contact) bool {
	for _, item := range cs {
		if item == c {
			return true
		}
	}
	return false
}
