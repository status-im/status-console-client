package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"
)

// Chat is an interface defining basic methods to receive and send messages.
type Chat interface {
	// Subscribe listens to new messages.
	Subscribe(ctx context.Context, messages chan<- *Message, options SubscribeOptions) (*Subscription, error)

	// Send sends a message to the network.
	// Identity is required as the protocol requires
	// all messages to be signed.
	Send(ctx context.Context, data []byte, options SendOptions) ([]byte, error)

	// Request retrieves historic messages.
	Request(ctx context.Context, params RequestOptions) error
}

// Message contains a decoded message payload
// and some additional fields that we learnt
// about the message.
type Message struct {
	Decoded   StatusMessage
	SigPubKey *ecdsa.PublicKey
	Hash      []byte
}

// RequestOptions is a list of params required
// to request for historic messages.
type RequestOptions struct {
	Limit int
	From  int64
	To    int64

	ChatName  string           // for public chats
	Recipient *ecdsa.PublicKey // for private chats
}

// Validate verifies that the given request options are valid.
func (o RequestOptions) Validate() error {
	if o == (RequestOptions{}) {
		return errors.New("empty options")
	}
	if o.ChatName == "" && o.Recipient == nil {
		return errors.New("field ChatName or Recipient is required")
	}
	if o.ChatName != "" && o.Recipient != nil {
		return errors.New("field ChatName and Recipient both set")
	}
	return nil
}

// IsPublic returns true if RequestOptions are for a public chat.
func (o RequestOptions) IsPublic() bool { return o.ChatName != "" }

// DefaultRequestOptions returns default options returning messages
// from the last 24 hours.
func DefaultRequestOptions() RequestOptions {
	return RequestOptions{
		From:  time.Now().Add(-24 * time.Hour).Unix(),
		To:    time.Now().Unix(),
		Limit: 1000,
	}
}

// SubscribeOptions are options for Chat.Subscribe method.
type SubscribeOptions struct {
	Identity *ecdsa.PrivateKey // for private chats
	ChatName string            // for public chats
}

// Validate vierifies that the given options are valid.
func (o SubscribeOptions) Validate() error {
	if o == (SubscribeOptions{}) {
		return errors.New("empty options")
	}
	if o.Identity != nil && o.ChatName != "" {
		return errors.New("fields Identity and ChatName both set")
	}
	return nil
}

// IsPublic returns true if SubscribeOptions are for a public chat.
func (o SubscribeOptions) IsPublic() bool { return o.ChatName != "" }

// SendOptions are options for Chat.Send.
type SendOptions struct {
	Identity *ecdsa.PrivateKey

	ChatName  string           // for public chats
	Recipient *ecdsa.PublicKey // for private chats
}

// Validate verifies that the given options are valid.
func (o SendOptions) Validate() error {
	if o.Identity == nil {
		return errors.New("field Identity is required")
	}
	if o.ChatName == "" && o.Recipient == nil {
		return errors.New("field ChatName or Recipient is required")
	}
	if o.ChatName != "" && o.Recipient != nil {
		return errors.New("fields ChatName and Recipient both set")
	}
	return nil
}

// IsPublic returns true if SendOptions are for a public chat.
func (o SendOptions) IsPublic() bool { return o.ChatName != "" }
