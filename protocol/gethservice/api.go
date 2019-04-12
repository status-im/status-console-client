package gethservice

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-console-client/protocol/v1"
)

type PublicAPI struct {
	service *Service
}

type SubscribeParams struct {
	RecipientPubKey hexutil.Bytes `json:"recipientPubKey"` // public key hex-encoded
	PubChatName     string        `json:"pubChatName"`
}

func (api *PublicAPI) Messages(ctx context.Context, params SubscribeParams) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	adapterOptions := protocol.SubscribeOptions{
		ChatName: params.PubChatName, // no transformation required
	}

	if len(params.RecipientPubKey) > 0 {
		publicKey, err := crypto.UnmarshalPubkey(params.RecipientPubKey)
		if err != nil {
			return nil, err
		}
		adapterOptions.Recipient = publicKey
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

type SendParams struct {
	RecipientPubKey hexutil.Bytes `json:"recipientPubKey"` // public key hex-encoded
	PubChatName     string        `json:"pubChatName"`
}

func (api *PublicAPI) Send(ctx context.Context, data hexutil.Bytes, params SendParams) (hexutil.Bytes, error) {
	adapterOptions := protocol.SendOptions{
		ChatName: params.PubChatName, // no transformation required
	}

	if len(params.RecipientPubKey) > 0 {
		publicKey, err := crypto.UnmarshalPubkey(params.RecipientPubKey)
		if err != nil {
			return nil, err
		}
		adapterOptions.Recipient = publicKey
	}

	hash, err := api.service.protocol.Send(ctx, data, adapterOptions)
	return hexutil.Bytes(hash), err
}
