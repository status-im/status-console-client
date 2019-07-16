package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

//go:generate stringer -type=ChatType

// ChatType defines a type of a chat.
type ChatType int

func (c ChatType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, c)), nil
}

func (c *ChatType) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case fmt.Sprintf(`"%s"`, PublicChat):
		*c = PublicChat
	case fmt.Sprintf(`"%s"`, OneToOneChat):
		*c = OneToOneChat
	default:
		return fmt.Errorf("invalid ChatType: %s", data)
	}

	return nil
}

// Types of chats.
const (
	PublicChat ChatType = iota + 1
	OneToOneChat
	PrivateGroupChat
)

// ChatMember is a member of a group chat
type ChatMember struct {
	Admin     bool
	Joined    bool
	PublicKey *ecdsa.PublicKey `json:"-"`
}

// Chat is a single chat
type Chat struct {
	id        string
	publicKey *ecdsa.PublicKey

	Name string   `json:"name"`
	Type ChatType `json:"type"`
	// Creation time, is that been used?
	Timestamp int64 `json:"timestamp"`
	UpdatedAt int64 `json:"updatedAt"`
	// Soft delete flag
	Active bool `json:"active"`
	// The color of the chat, makes no sense outside a UI context
	Color string `json:"color"`
	// Clock value of the last message before chat has been deleted
	DeletedAtClockValue int64 `json:"deletedAtClockValue"`
	// Denormalized fields

	UnviewedMessageCount   int    `json:"unviewedMessageCount"`
	LastClockValue         int64  `json:"lastClockValue"`
	LastMessageContentType string `json:"lastMessageContentType"`
	LastMessageContent     string `json:"lastMessageContent"`
}

// CreateOneToOneChat creates a new private chat.
func CreateOneToOneChat(name, pubKeyHex string) (c Chat, err error) {
	pubKeyBytes, err := hexutil.Decode(pubKeyHex)
	if err != nil {
		return
	}

	c.id = hex.EncodeToString(pubKeyBytes)
	c.Name = name
	c.Type = OneToOneChat
	c.publicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)

	return
}

// CreatePublicChat creates a public room chat.
func CreatePublicChat(name string) Chat {
	return Chat{
		id:   name,
		Name: name,
		Type: PublicChat,
	}
}

// String returns a string representation of Chat.
func (c Chat) String() string {
	return c.Name
}

// Equal returns true if chats have same name and same type.
func (c Chat) Equal(other Chat) bool {
	return c.Name == other.Name && c.Type == other.Type
}

func (c Chat) ID() string                  { return c.id }
func (c Chat) PublicName() string          { return c.Name }
func (c Chat) PublicKey() *ecdsa.PublicKey { return c.publicKey }

func (c Chat) MarshalJSON() ([]byte, error) {
	type ChatAlias Chat

	item := struct {
		ChatAlias
		ID        string `json:"id"`
		PublicKey string `json:"public_key,omitempty"`
	}{
		ChatAlias: ChatAlias(c),
		ID:        c.ID(),
	}

	if c.PublicKey() != nil {
		item.PublicKey = encodePublicKeyAsString(c.PublicKey())
	}

	return json.Marshal(&item)
}

func (c *Chat) UnmarshalJSON(data []byte) error {
	type ChatAlias Chat

	var item struct {
		*ChatAlias
		ID        string `json:"id"`
		PublicKey string `json:"public_key,omitempty"`
	}

	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}

	item.ChatAlias.id = item.ID

	if len(item.PublicKey) > 2 {
		pubKey, err := hexutil.Decode(item.PublicKey)
		if err != nil {
			return err
		}

		item.ChatAlias.publicKey, err = crypto.UnmarshalPubkey(pubKey)
		if err != nil {
			return err
		}
	}

	*c = *(*Chat)(item.ChatAlias)

	return nil
}

// encodePublicKeyAsString encodes a public key as a string.
// It starts with 0x to indicate it's hex encoding.
func encodePublicKeyAsString(pubKey *ecdsa.PublicKey) string {
	return hexutil.Encode(crypto.FromECDSAPub(pubKey))
}

func isPrivateChat(c Chat) bool {
	return c.PublicKey() != nil
}

func isPublicChat(c Chat) bool {
	return c.PublicKey() == nil
}
