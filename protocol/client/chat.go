package client

import (
	"crypto/ecdsa"
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
	// The id of the chat
	ID   string   `json:"id"`
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
	DeletedAtClockValue int64            `json:"deletedAtClockValue"`
	PublicKey           *ecdsa.PublicKey `json:"-"`
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

	c.Name = name
	c.Type = OneToOneChat
	c.PublicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)

	return
}

// CreatePublicChat creates a public room chat.
func CreatePublicChat(name string) Chat {
	return Chat{
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

func (c Chat) MarshalJSON() ([]byte, error) {
	type ChatAlias Chat

	item := struct {
		ChatAlias
		PublicKey string `json:"public_key,omitempty"`
	}{
		ChatAlias: ChatAlias(c),
	}

	if c.PublicKey != nil {
		item.PublicKey = EncodePublicKeyAsString(c.PublicKey)
	}

	return json.Marshal(&item)
}

func (c *Chat) UnmarshalJSON(data []byte) error {
	type ChatAlias Chat

	var item struct {
		*ChatAlias
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

		item.ChatAlias.PublicKey, err = crypto.UnmarshalPubkey(pubKey)
		if err != nil {
			return err
		}
	}

	*c = *(*Chat)(item.ChatAlias)

	return nil
}

// EncodePublicKeyAsString encodes a public key as a string.
// It starts with 0x to indicate it's hex encoding.
func EncodePublicKeyAsString(pubKey *ecdsa.PublicKey) string {
	return hexutil.Encode(crypto.FromECDSAPub(pubKey))
}
