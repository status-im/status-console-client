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

// MessagesParams is an object with JSON-serializable parameters
// for Messages method.
type MessagesParams struct {
	Contact
}

// SendParams is an object with JSON-serializable parameters for Send method.
type SendParams struct {
	Contact
}

// RequestParams is an object with JSON-serializable parameters for Request method.
type RequestParams struct {
	Contact
	Limit int   `json:"limit"`
	From  int64 `json:"from"`
	To    int64 `json:"to"`
}

// Contact
type Contact struct {
	Name      string        `json:"name"`
	PublicKey hexutil.Bytes `json:"key"`
}

func parseContact(c Contact) (client.Contact, error) {
	if len(c.PublicKey) != 0 {
		c, err := client.CreateContactPrivate(c.Name, c.PublicKey.String(), client.ContactAdded)
		if err != nil {
			return c, err
		}
	}
	return client.CreateContactPublicRoom(c.Name, client.ContactAdded), nil
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
			ChatName: params.Name, // no transformation required
		},
	}

	adapterOptions.Recipient, err = unmarshalPubKey(params.PublicKey)
	if err != nil {
		return nil, err
	}

	messages := make(chan *protocol.StatusMessage, 100)
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
			ChatName: params.Name, // no transformation required
		},
	}

	adapterOptions.Recipient, err = unmarshalPubKey(params.PublicKey)
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
	c, err := parseContact(params.Contact)
	if err != nil {
		return err
	}
	options := protocol.RequestOptions{
		Limit: params.Limit,
		From:  params.From,
		To:    params.To,
	}
	return api.service.messenger.Request(ctx, c, options)
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
		api.broadcaster = newBroadcaster(api.service.messenger)
	}

	// Subscription needs to be created
	// before any events are delivered.
	sub := api.broadcaster.Subscribe(contact)

	err := api.service.messenger.Join(ctx, contact)
	if err != nil {
		api.broadcaster.Unsubscribe(sub)
		return nil, err
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		defer func() {
			err := api.service.messenger.Leave(contact)
			if err != nil {
				log.Printf("failed to leave chat for '%s' contact", contact)
			}
		}()
		defer api.broadcaster.Unsubscribe(sub)

		for {
			select {
			case e := <-sub:
				if err := notifier.Notify(rpcSub.ID, e); err != nil {
					log.Printf("failed to notify %s about new message", rpcSub.ID)
				}
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

// AddContact will ensure that contact is added to messenger database and new stream spawned for a contact if needed.
func (api *PublicAPI) AddContact(ctx context.Context, contact Contact) (err error) {
	if api.service.messenger == nil {
		return ErrMessengerNotSet
	}
	c, err := parseContact(contact)
	if err != nil {
		return err
	}
	err = api.service.messenger.AddContact(c)
	if err != nil {
		return err
	}
	return api.service.messenger.Join(ctx, c)
}

// SendToContact send payload to specified contact. Contact should be added before sending message, otherwise error will
// be received.
func (api *PublicAPI) SendToContact(ctx context.Context, contact Contact, payload string) (err error) {
	if api.service.messenger == nil {
		return ErrMessengerNotSet
	}
	c, err := parseContact(contact)
	if err != nil {
		return err
	}
	return api.service.messenger.Send(c, []byte(payload))
}

// ReadContactMessages read contact messages starting from offset.
// To read all offset should be zero. To read only new set offset to total number of previously read messages.
func (api *PublicAPI) ReadContactMessages(ctx context.Context, contact Contact, offset int64) (rst []*protocol.Message, err error) {
	if api.service.messenger == nil {
		return nil, ErrMessengerNotSet
	}
	c, err := parseContact(contact)
	if err != nil {
		return nil, err
	}
	return api.service.messenger.Messages(c, offset)
}

func unmarshalPubKey(b hexutil.Bytes) (*ecdsa.PublicKey, error) {
	if len(b) == 0 {
		return nil, nil
	}
	return crypto.UnmarshalPubkey(b)
}
