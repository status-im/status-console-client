package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"
)

type Chat interface {
	Subscribe(
		ctx context.Context,
		messages chan<- *Message,
		options SubscribeOptions,
	) (*Subscription, error)

	// Send sends a message to the network.
	// Identity is required as the protocol requires
	// all messages to be signed.
	Send(
		ctx context.Context,
		data []byte,
		options SendOptions,
	) ([]byte, error)

	Request(ctx context.Context, params RequestOptions) error
}

// Message contains a decoded message payload
// and some additional fields that we learnt
// about the message.
type Message struct {
	Decoded   StatusMessage
	Hash      []byte
	SigPubKey *ecdsa.PublicKey
}

// type receivedMessageGob struct {
// 	Decoded   StatusMessage
// 	Hash      []byte
// 	SigPubKey []byte
// }

// func (m *ReceivedMessage) GobEncode() ([]byte, error) {
// 	val := receivedMessageGob{
// 		Decoded: m.Decoded,
// 		Hash:    m.Hash,
// 	}

// 	if m.SigPubKey != nil {
// 		val.SigPubKey = crypto.FromECDSAPub(m.SigPubKey)
// 	}

// 	fmt.Printf("GobEncode: %+v", val)

// 	var buf bytes.Buffer

// 	enc := gob.NewEncoder(&buf)
// 	if err := enc.Encode(&val); err != nil {
// 		return nil, err
// 	}

// 	return buf.Bytes(), nil
// }

// func (m *ReceivedMessage) GobDecode(data []byte) error {
// 	var val receivedMessageGob

// 	buf := bytes.NewBuffer(data)
// 	enc := gob.NewDecoder(buf)
// 	if err := enc.Decode(&val); err != nil {
// 		return err
// 	}

// 	var err error

// 	m.Decoded = val.Decoded
// 	m.Hash = val.Hash
// 	if val.SigPubKey != nil {
// 		m.SigPubKey, err = crypto.UnmarshalPubkey(val.SigPubKey)
// 	}

// 	fmt.Printf("GobDecode: %+v", val)

// 	return err
// }

// RequestOptions is a list of params required
// to request for historic messages.
type RequestOptions struct {
	ChatName  string           // for public chats
	Recipient *ecdsa.PublicKey // for private chats
	Limit     int
	From      int64
	To        int64
}

func (o RequestOptions) Validate() error {
	if o == (RequestOptions{}) {
		return errors.New("empty request messages options")
	}
	if o.ChatName == "" && o.Recipient == nil {
		return errors.New("chat name or recipient is required")
	}
	if o.ChatName != "" && o.Recipient != nil {
		return errors.New("chat name and recipient both set")
	}
	return nil
}

func (o RequestOptions) IsPublic() bool { return o.ChatName != "" }

func DefaultRequestOptions() RequestOptions {
	return RequestOptions{
		From:  time.Now().Add(-24 * time.Hour).Unix(),
		To:    time.Now().Unix(),
		Limit: 1000,
	}
}

type SubscribeOptions struct {
	Identity *ecdsa.PrivateKey // for private chats
	ChatName string            // for public chats
}

func (o SubscribeOptions) Validate() error {
	if o == (SubscribeOptions{}) {
		return errors.New("empty subscribe options")
	}
	if o.Identity != nil && o.ChatName != "" {
		return errors.New("identity and chat name both set")
	}
	return nil
}

func (o SubscribeOptions) IsPublic() bool { return o.ChatName != "" }

type SendOptions struct {
	Identity  *ecdsa.PrivateKey
	ChatName  string           // for public chats
	Recipient *ecdsa.PublicKey // for private chats
}

func (o SendOptions) Validate() error {
	if o.Identity == nil {
		return errors.New("identity is required")
	}
	if o.ChatName == "" && o.Recipient == nil {
		return errors.New("chat name or recipient is required")
	}
	if o.ChatName != "" && o.Recipient != nil {
		return errors.New("chat name and recipient both set")
	}
	return nil
}

func (o SendOptions) IsPublic() bool { return o.ChatName != "" }
