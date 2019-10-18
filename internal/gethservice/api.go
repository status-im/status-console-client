// +build geth !nimbus

package gethservice

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	status "github.com/status-im/status-protocol-go"
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
	status.Chat
}

// SendParams is an object with JSON-serializable parameters for Send method.
type SendParams struct {
	status.Chat
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
func (api *PublicAPI) Send(ctx context.Context, chatID string, payload string) ([]hexutil.Bytes, error) {
	if api.service.messenger == nil {
		return nil, ErrMessengerNotSet
	}
	ids, err := api.service.messenger.Send(ctx, chatID, []byte(payload))
	if err != nil {
		return nil, err
	}
	result := make([]hexutil.Bytes, len(ids))
	for idx, id := range ids {
		result[idx] = id
	}
	return result, nil
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
