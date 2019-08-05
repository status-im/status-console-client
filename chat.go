package main

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	status "github.com/status-im/status-protocol-go"
)

// CreateOneToOneChat creates a new private chat.
func CreateOneToOneChat(name, pubKeyHex string) (c *status.Chat, err error) {
	pubKeyBytes, err := hexutil.Decode(pubKeyHex)
	if err != nil {
		return
	}

	c = &status.Chat{
		ID:       pubKeyHex,
		Name:     name,
		ChatType: status.ChatTypeOneToOne,
	}
	c.PublicKey, err = crypto.UnmarshalPubkey(pubKeyBytes)

	return
}

// CreatePublicChat creates a public room chat.
func CreatePublicChat(name string) *status.Chat {
	return &status.Chat{
		ID:       name,
		Name:     name,
		ChatType: status.ChatTypePublic,
	}
}
