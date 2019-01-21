package protocol

import (
	"context"
	"crypto/ecdsa"
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

	// TODO: RequestMessagesParams is Whisper specific.
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
	Decoded   StatusMessage
	SigPubKey *ecdsa.PublicKey
}
