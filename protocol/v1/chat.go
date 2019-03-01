package protocol

import (
	"context"
	"crypto/ecdsa"
)

// Chat provides an interface to interact with any chat.
type Chat interface {
	PublicChat
	PrivateChat
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
	) ([]byte, error)

	// TODO: RequestMessagesParams is Whisper specific.
	RequestPublicMessages(
		ctx context.Context,
		chatName string,
		params RequestMessagesParams,
	) error
}

// PrivateChat provides an interface to interact with private chats.
type PrivateChat interface {
	SubscribePrivateChat(
		ctx context.Context,
		identity *ecdsa.PrivateKey,
		in chan<- *ReceivedMessage,
	) (*Subscription, error)

	SendPrivateMessage(
		ctx context.Context,
		recipient *ecdsa.PublicKey,
		data []byte,
		identity *ecdsa.PrivateKey,
	) ([]byte, error)

	RequestPrivateMessages(ctx context.Context, params RequestMessagesParams) error
}

// ReceivedMessage contains a decoded message payload
// and some additional fields that we learnt
// about the message.
type ReceivedMessage struct {
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

// RequestMessagesParams is a list of params required
// to get historic messages.
type RequestMessagesParams struct {
	Limit int
	From  int64
	To    int64
}
