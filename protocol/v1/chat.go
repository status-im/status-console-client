package protocol

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

// Chat provides an interface to interact with any chat.
type Chat interface {
	PublicChat
}

// PublicChat provides an interface to interact with public chats.
type PublicChat interface {
	SubscribePublicChat(
		ctx context.Context,
		name string,
		in chan<- *ReceivedMessage,
	) (*Subscription, error)

	// SendPublicMessages sends a message to a public chat.
	// Identity is required as the protocol requires
	// all messages to be signed.
	SendPublicMessage(
		ctx context.Context,
		chatName string,
		data []byte,
		identity *ecdsa.PrivateKey,
	) (string, error)

	RequestPublicMessages(
		ctx context.Context,
		chatName string,
		params RequestMessagesParams,
	) error
}

// ReceivedMessage contains a decoded message payload
// and some additional fields that we learnt
// about the message.
type ReceivedMessage struct {
	Decoded StatusMessage
	Src     []byte
}

// SrcPubKey returns a public key of the source of the message.
func (r ReceivedMessage) SrcPubKey() (*ecdsa.PublicKey, error) {
	return crypto.UnmarshalPubkey(r.Src)
}
