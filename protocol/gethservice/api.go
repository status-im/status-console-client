package gethservice

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	// ErrProtocolNotSet tells that the protocol was not set in the Service.
	ErrProtocolNotSet = errors.New("protocol is not set")
)

// PublicAPI provides an JSON-RPC API to interact with
// the Status Messaging Protocol through a geth node.
type PublicAPI struct {
	service *Service
}

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
				if err := notifier.Notify(rpcSub.ID, m.Decoded); err != nil {
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

// SendParams is an object with JSON-serializable parameters for Send method.
type SendParams struct {
	ChatParams
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

// RequestParams is an object with JSON-serializable parameters for Request method.
type RequestParams struct {
	ChatParams
	Limit int   `json:"limit"`
	From  int64 `json:"from"`
	To    int64 `json:"to"`
}

// Request send a request for historic messages matching the provided RequestParams.
func (api *PublicAPI) Request(ctx context.Context, params RequestParams) (err error) {
	if api.service.protocol == nil {
		return ErrProtocolNotSet
	}

	adapterOptions := protocol.RequestOptions{
		ChatOptions: protocol.ChatOptions{
			ChatName: params.PubChatName, // no transformation required
		},
		Limit: params.Limit,
		From:  params.From,
		To:    params.To,
	}

	adapterOptions.Recipient, err = unmarshalPubKey(params.RecipientPubKey)
	if err != nil {
		return
	}

	return api.service.protocol.Request(ctx, adapterOptions)
}

func unmarshalPubKey(b hexutil.Bytes) (*ecdsa.PublicKey, error) {
	if len(b) == 0 {
		return nil, nil
	}
	return crypto.UnmarshalPubkey(b)
}
