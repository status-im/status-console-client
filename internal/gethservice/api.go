package gethservice

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// ErrProtocolNotSet tells that the protocol was not set in the Service.
	ErrProtocolNotSet = errors.New("protocol is not set")
	// ErrMessengerNotSet tells that the messenger was not set in the Service.
	ErrMessengerNotSet = errors.New("messenger is not set")
)

// MessagesParams is an object with JSON-serializable parameters
// for Messages method.
type MessagesParams struct {
	Chat
}

// SendParams is an object with JSON-serializable parameters for Send method.
type SendParams struct {
	Chat
}

// RequestParams is an object with JSON-serializable parameters for Request method.
type RequestParams struct {
	Chat
	Limit int   `json:"limit"`
	From  int64 `json:"from"`
	To    int64 `json:"to"`
}

// Chat
type Chat struct {
	chatID     string
	publicName string
	publicKey  *ecdsa.PublicKey
}

func (c Chat) ID() string                  { return c.chatID }
func (c Chat) PublicName() string          { return c.publicName }
func (c Chat) PublicKey() *ecdsa.PublicKey { return c.publicKey }

func (c Chat) MarshalJSON() ([]byte, error) {
	type ChatAlias Chat

	item := struct {
		ChatAlias
		ID         string `json:"id"`
		PublicName string `json:"public_name,omitempty"`
		PublicKey  string `json:"public_key,omitempty"`
	}{
		ChatAlias:  ChatAlias(c),
		ID:         c.ID(),
		PublicName: c.PublicName(),
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
		ID         string `json:"id"`
		PublicName string `json:"public_name,omitempty"`
		PublicKey  string `json:"public_key,omitempty"`
	}

	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}

	item.ChatAlias.chatID = item.ID
	item.ChatAlias.publicName = item.PublicName

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

// PublicAPI provides an JSON-RPC API to interact with
// the Status Messaging Protocol through a geth node.
type PublicAPI struct {
	service *Service
}

func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		service: s,
	}
}

// Send sends payload to specified chat.
// Chat should be added before sending message,
// otherwise error will be received.
func (api *PublicAPI) Send(ctx context.Context, chat Chat, payload string) (hexutil.Bytes, error) {
	if api.service.messenger == nil {
		return nil, ErrMessengerNotSet
	}

	return api.service.messenger.Send(ctx, chat, []byte(payload))
}

// Request sends a request for historic messages matching the provided RequestParams.
// func (api *PublicAPI) Request(ctx context.Context, params RequestParams) (err error) {
// 	if api.service.messenger == nil {
// 		return ErrMessengerNotSet
// 	}
// 	c, err := parseChat(params.Chat)
// 	if err != nil {
// 		return err
// 	}
// 	options := protocol.RequestOptions{
// 		Limit: params.Limit,
// 		From:  params.From,
// 		To:    params.To,
// 	}
// 	return api.service.messenger.Request(ctx, c, options)
// }

// Messages is a high-level subscription-based RPC method.
// It joins a chat for selected chat and streams
// events for that chat.
// func (api *PublicAPI) Messages(ctx context.Context, chat Chat) (*rpc.Subscription, error) {
// 	notifier, supported := rpc.NotifierFromContext(ctx)
// 	if !supported {
// 		return nil, rpc.ErrNotificationsUnsupported
// 	}

// 	if api.service.messenger == nil {
// 		return nil, ErrMessengerNotSet
// 	}

// 	err := api.service.messenger.Join(ctx, chat)
// 	if err != nil {
// 		api.broadcaster.Unsubscribe(sub)
// 		return nil, err
// 	}

// 	rpcSub := notifier.CreateSubscription()

// 	go func() {
// 		defer func() {
// 			err := api.service.messenger.Leave(chat)
// 			if err != nil {
// 				log.Printf("failed to leave chat for '%s' chat", chat)
// 			}
// 		}()
// 		defer api.broadcaster.Unsubscribe(sub)

// 		for {
// 			select {
// 			case e := <-sub:
// 				if err := notifier.Notify(rpcSub.ID, e); err != nil {
// 					log.Printf("failed to notify %s about new message", rpcSub.ID)
// 				}
// 			case err := <-rpcSub.Err():
// 				if err != nil {
// 					log.Printf("RPC subscription errored: %v", err)
// 				}
// 				return
// 			case <-notifier.Closed():
// 				log.Printf("notifier closed")
// 				return
// 			case <-ctx.Done():
// 				log.Printf("context is canceled: %v", ctx.Err())
// 				return
// 			}
// 		}
// 	}()

// 	return rpcSub, nil
// }

// RequestAll sends a request for messages for all subscribed chats.
// If newest is set to true, it requests the most recent messages.
// Otherwise, it requests older messages than already downloaded.
// func (api *PublicAPI) RequestAll(ctx context.Context, newest bool) error {
// 	return api.service.messenger.RequestAll(ctx, newest)
// }

// AddChat will ensure that chat is added to messenger database and new stream spawned for a chat if needed.
// func (api *PublicAPI) AddChat(ctx context.Context, chat Chat) (err error) {
// 	if api.service.messenger == nil {
// 		return ErrMessengerNotSet
// 	}
// 	c, err := parseChat(chat)
// 	if err != nil {
// 		return err
// 	}
// 	err = api.service.messenger.AddChat(c)
// 	if err != nil {
// 		return err
// 	}
// 	return api.service.messenger.Join(ctx, c)
// }

// ReadContactMessages read contact messages starting from offset.
// To read all offset should be zero. To read only new set offset to total number of previously read messages.
// func (api *PublicAPI) ReadChatMessages(ctx context.Context, chat Chat, offset int64) (rst []*protocol.Message, err error) {
// 	if api.service.messenger == nil {
// 		return nil, ErrMessengerNotSet
// 	}
// 	c, err := parseChat(chat)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return api.service.messenger.Messages(c, offset)
// }

// encodePublicKeyAsString encodes a public key as a string.
// It starts with 0x to indicate it's hex encoding.
func encodePublicKeyAsString(pubKey *ecdsa.PublicKey) string {
	return hexutil.Encode(crypto.FromECDSAPub(pubKey))
}
