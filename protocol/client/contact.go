package client

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

//go:generate stringer -type=ContactType

// ContactType defines a type of a contact.
type ContactType int

// ContactState defines state of the contact.
type ContactState int

func (c ContactType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, c)), nil
}

func (c *ContactType) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case fmt.Sprintf(`"%s"`, ContactPublicRoom):
		*c = ContactPublicRoom
	case fmt.Sprintf(`"%s"`, ContactPublicKey):
		*c = ContactPublicKey
	default:
		return fmt.Errorf("invalid ContactType: %s", data)
	}

	return nil
}

// Types of contacts.
const (
	ContactPublicRoom ContactType = iota + 1
	ContactPublicKey

	// ContactAdded default level. Added or confirmed by user.
	ContactAdded ContactState = iota
	// ContactNew contact got connected to us and waits for being added or blocked.
	ContactNew
	// Messages of the blocked contact must be discarded (or atleast not visible to the user)
	ContactBlocked
)

// Contact is a single contact which has a type and name.
type Contact struct {
	Name      string           `json:"name"`
	Type      ContactType      `json:"type"`
	State     ContactState     `json:"state"`
	Topic     string           `json:"topic"`
	PublicKey *ecdsa.PublicKey `json:"-"`
}

// String returns a string representation of Contact.
func (c Contact) String() string {
	return c.Name
}

func (c Contact) MarshalJSON() ([]byte, error) {
	type ContactAlias Contact

	item := struct {
		ContactAlias
		PublicKey string `json:"public_key,omitempty"`
	}{
		ContactAlias: ContactAlias(c),
	}

	if c.PublicKey != nil {
		item.PublicKey = hexutil.Encode(crypto.FromECDSAPub(c.PublicKey))
	}

	return json.Marshal(&item)
}

func (c *Contact) UnmarshalJSON(data []byte) error {
	type ContactAlias Contact

	var item struct {
		*ContactAlias
		PublicKey string `json:"public_key,omitempty"`
	}

	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}

	if len(item.PublicKey) > 2 {
		pubKey, err := hexutil.Decode(item.PublicKey)
		if err != nil {
			return err
		}

		item.ContactAlias.PublicKey, err = crypto.UnmarshalPubkey(pubKey)
		if err != nil {
			return err
		}
	}

	*c = *(*Contact)(item.ContactAlias)

	return nil
}

// ContactWithPublicKey creates a new private contact.
func ContactWithPublicKey(name, pubKeyHex string) (c Contact, err error) {
	c.Name = name
	c.Type = ContactPublicKey
	c.Topic = DefaultPrivateTopic()
	pubKeyBytes, err := hexutil.Decode(pubKeyHex)
	if err != nil {
		return
	}

	c.PublicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)
	return
}
