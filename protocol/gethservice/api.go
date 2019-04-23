package gethservice

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	// ErrProtocolNotSet tells that the protocol was not set in the Service.
	ErrProtocolNotSet = errors.New("protocol is not set")
	// ErrMessengerNotSet tells that the messenger was not set in the Service.
	ErrMessengerNotSet = errors.New("messenger is not set")
)

// ChatParams are chat specific options.
type ChatParams struct {
	RecipientPubKey hexutil.Bytes `json:"recipientPubKey"` // public key hex-encoded
	PubChatName     string        `json:"pubChatName"`
}

// MessagesParams is an object with JSON-serializable parameters
// for Messages method.
type MessagesParams struct {
	ChatParams
}

// SendParams is an object with JSON-serializable parameters for Send method.
type SendParams struct {
	ChatParams
}

// RequestParams is an object with JSON-serializable parameters for Request method.
type RequestParams struct {
	ChatParams
	Limit int   `json:"limit"`
	From  int64 `json:"from"`
	To    int64 `json:"to"`
}

// Contact
type Contact struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// PublicAPI provides an JSON-RPC API to interact with
// the Status Messaging Protocol through a geth node.
type PublicAPI struct {
	service     *Service
	broadcaster *broadcaster
}

func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		service: s,
	}
}

// Messages creates an RPC subscription which delivers received messages.
func (api *PublicAPI) Messages(ctx context.Context, params MessagesParams) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	if api.service.protocol == nil {
		return nil, ErrProtocolNotSet
	}

	var err error

	adapterOptions := protocol.SubscribeOptions{
		ChatOptions: protocol.ChatOptions{
			ChatName: params.PubChatName, // no transformation required
		},
	}

	adapterOptions.Recipient, err = unmarshalPubKey(params.RecipientPubKey)
	if err != nil {
		return nil, err
	}

	messages := make(chan *protocol.Message, 100)
	sub, err := api.service.protocol.Subscribe(ctx, messages, adapterOptions)
	if err != nil {
		log.Printf("failed to subscribe to the protocol: %v", err)
		return nil, err
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		defer sub.Unsubscribe()

		for {
			select {
			case m := <-messages:
				if err := notifier.Notify(rpcSub.ID, m); err != nil {
					log.Printf("failed to notify %s about new message", rpcSub.ID)
				}
			case <-sub.Done():
				if err := sub.Err(); err != nil {
					log.Printf("subscription to adapter errored: %v", err)
				}
				return
			case err := <-rpcSub.Err():
				if err != nil {
					log.Printf("RPC subscription errored: %v", err)
				}
				return
			case <-notifier.Closed():
				log.Printf("notifier closed")
				return
			}
		}
	}()

	return rpcSub, nil
}

// Send sends a message to the network.
func (api *PublicAPI) Send(ctx context.Context, data hexutil.Bytes, params SendParams) (hexutil.Bytes, error) {
	if api.service.protocol == nil {
		return nil, ErrProtocolNotSet
	}

	var err error

	adapterOptions := protocol.SendOptions{
		ChatOptions: protocol.ChatOptions{
			ChatName: params.PubChatName, // no transformation required
		},
	}

	adapterOptions.Recipient, err = unmarshalPubKey(params.RecipientPubKey)
	if err != nil {
		return nil, err
	}

	return api.service.protocol.Send(ctx, data, adapterOptions)
}

// Request sends a request for historic messages matching the provided RequestParams.
func (api *PublicAPI) Request(ctx context.Context, params RequestParams) (err error) {
	if api.service.protocol == nil {
		return ErrProtocolNotSet
	}

	adapterOptions := protocol.RequestOptions{
		Chats: []protocol.ChatOptions{
			protocol.ChatOptions{
				ChatName: params.PubChatName, // no transformation required
			},
		},
		Limit: params.Limit,
		From:  params.From,
		To:    params.To,
	}

	adapterOptions.Chats[0].Recipient, err = unmarshalPubKey(params.RecipientPubKey)
	if err != nil {
		return
	}

	return api.service.protocol.Request(ctx, adapterOptions)
}

// Chat is a high-level subscription-based RPC method.
// It joins a chat for selected contact and streams
// events for that chat.
func (api *PublicAPI) Chat(ctx context.Context, contact client.Contact) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	if api.service.messenger == nil {
		return nil, ErrMessengerNotSet
	}

	// Create a broadcaster instance.
	// TODO: move it.
	if api.broadcaster == nil {
		api.broadcaster = newBroadcaster(api.service.messenger.Events())
	}

	// Subscription needs to be created
	// before any events are delivered.
	sub := api.broadcaster.Subscribe(contact)

	chat, err := api.service.messenger.Join(ctx, contact)
	if err != nil {
		api.broadcaster.Unsubscribe(sub)
		return nil, err
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		defer api.service.messenger.Leave(contact)
		defer api.broadcaster.Unsubscribe(sub)

		for {
			select {
			case e := <-sub:
				if err := notifier.Notify(rpcSub.ID, e); err != nil {
					log.Printf("failed to notify %s about new message", rpcSub.ID)
				}
			case <-chat.Done():
				if err := chat.Err(); err != nil {
					log.Printf("chat errored: %v", err)
				}
				return
			case err := <-rpcSub.Err():
				if err != nil {
					log.Printf("RPC subscription errored: %v", err)
				}
				return
			case <-notifier.Closed():
				log.Printf("notifier closed")
				return
			case <-ctx.Done():
				log.Printf("context is canceled: %v", ctx.Err())
				return
			}
		}
	}()

	return rpcSub, nil
}

// RequestAll sends a request for messages for all subscribed chats.
// If newest is set to true, it requests the most recent messages.
// Otherwise, it requests older messages than already downloaded.
func (api *PublicAPI) RequestAll(ctx context.Context, newest bool) error {
	return api.service.messenger.RequestAll(ctx, newest)
}

func unmarshalPubKey(b hexutil.Bytes) (*ecdsa.PublicKey, error) {
	if len(b) == 0 {
		return nil, nil
	}
	return crypto.UnmarshalPubkey(b)
}
