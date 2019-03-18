package adapters

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/t/helpers"

	whisper "github.com/status-im/whisper/whisperv6"
)

// WhisperServiceAdapter is an adapter for Whisper service
// the implements Chat interface.
type WhisperServiceAdapter struct {
	node *node.StatusNode
	shh  *whisper.Whisper

	selectedMailServerEnode string
}

// WhisperServiceAdapter must implement Chat interface.
var _ protocol.Chat = (*WhisperServiceAdapter)(nil)

// NewWhisperServiceAdapter returns a new WhisperServiceAdapter.
func NewWhisperServiceAdapter(node *node.StatusNode, shh *whisper.Whisper) *WhisperServiceAdapter {
	return &WhisperServiceAdapter{
		node: node,
		shh:  shh,
	}
}

// Subscribe subscribes to a public chat using the Whisper service.
func (a *WhisperServiceAdapter) Subscribe(
	ctx context.Context,
	in chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter, err := a.createFilter(options)
	if err != nil {
		return nil, err
	}

	filterID, err := a.shh.Subscribe(filter)
	if err != nil {
		return nil, err
	}

	subWhisper := newWhisperSubscription(a.shh, filterID)
	sub := protocol.NewSubscription()

	go func() {
		defer subWhisper.Unsubscribe() // nolint: errcheck

		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				messages, err := subWhisper.Messages()
				if err != nil {
					sub.Cancel(err)
					return
				}

				sort.Slice(messages, func(i, j int) bool {
					return messages[i].Decoded.Clock < messages[j].Decoded.Clock
				})

				for _, m := range messages {
					in <- m
				}
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

func (a *WhisperServiceAdapter) createFilter(opts protocol.SubscribeOptions) (*whisper.Filter, error) {
	filter := whisper.Filter{
		PoW:      0,
		AllowP2P: true,
	}

	if opts.IsPublic() {
		symKeyID, err := a.shh.AddSymKeyFromPassword(opts.ChatName)
		if err != nil {
			return nil, err
		}
		symKey, err := a.shh.GetSymKey(symKeyID)
		if err != nil {
			return nil, err
		}
		topic, err := PublicChatTopic(opts.ChatName)
		if err != nil {
			return nil, err
		}

		filter.KeySym = symKey
		filter.Topics = append(filter.Topics, topic[:])
	} else {
		filter.KeyAsym = opts.Identity

		topic, err := PrivateChatTopic()
		if err != nil {
			return nil, err
		}

		filter.Topics = append(filter.Topics, topic[:])
	}

	return &filter, nil
}

// Send sends a new message using the Whisper service.
func (a *WhisperServiceAdapter) Send(
	ctx context.Context,
	data []byte,
	options protocol.SendOptions,
) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	newMessage, err := a.createNewMessage(data, options)
	if err != nil {
		return nil, err
	}

	// Only public Whisper API implements logic to send messages.
	shhAPI := whisper.NewPublicWhisperAPI(a.shh)
	return shhAPI.Post(ctx, newMessage)
}

func (a *WhisperServiceAdapter) createNewMessage(data []byte, options protocol.SendOptions) (message whisper.NewMessage, err error) {
	// TODO: add cache
	keyID, err := a.shh.AddKeyPair(options.Identity)
	if err != nil {
		return
	}

	message = createWhisperNewMessage(data, keyID)

	if options.IsPublic() {
		symKeyID, err := a.shh.AddSymKeyFromPassword(options.ChatName)
		if err != nil {
			return message, err
		}
		message.SymKeyID = symKeyID

		topic, err := PublicChatTopic(options.ChatName)
		if err != nil {
			return message, err
		}
		message.Topic = topic
	} else {
		message.PublicKey = crypto.FromECDSAPub(options.Recipient)

		topic, err := PrivateChatTopic()
		if err != nil {
			return message, err
		}
		message.Topic = topic
	}

	return
}

// Request requests messages from mail servers.
func (a *WhisperServiceAdapter) Request(ctx context.Context, options protocol.RequestOptions) error {
	if err := options.Validate(); err != nil {
		return err
	}

	// TODO: remove from here. MailServerEnode must be provided in the params.
	enode, err := a.selectAndAddMailServer()
	if err != nil {
		return err
	}

	// TODO: handle cursor from the response.
	resp, err := a.requestMessages(ctx, enode, options)
	if err != nil {
		return err
	}

	return resp.Error
}

func (a *WhisperServiceAdapter) selectAndAddMailServer() (string, error) {
	if a.selectedMailServerEnode != "" {
		return a.selectedMailServerEnode, nil
	}

	config := a.node.Config()
	enode := randomItem(config.ClusterConfig.TrustedMailServers)
	errCh := helpers.WaitForPeerAsync(
		a.node.GethNode().Server(),
		enode,
		p2p.PeerEventTypeAdd,
		time.Second*5,
	)
	if err := a.node.AddPeer(enode); err != nil {
		return "", err
	}

	err := <-errCh
	if err != nil {
		err = fmt.Errorf("failed to add mail server %s: %v", enode, err)
	} else {
		a.selectedMailServerEnode = enode
	}

	return enode, err
}

func (a *WhisperServiceAdapter) requestMessages(ctx context.Context, enode string, params protocol.RequestOptions) (resp shhext.MessagesResponse, err error) {
	shhextService, err := a.node.ShhExtService()
	if err != nil {
		return
	}
	shhextAPI := shhext.NewPublicAPI(shhextService)

	req, err := a.createMessagesRequest(enode, params)
	if err != nil {
		return
	}

	return shhextAPI.RequestMessagesSync(shhext.RetryConfig{
		BaseTimeout: time.Second * 10,
		StepTimeout: time.Second,
		MaxRetries:  3,
	}, req)
}

func (a *WhisperServiceAdapter) createMessagesRequest(
	enode string,
	params protocol.RequestOptions,
) (req shhext.MessagesRequest, err error) {
	mailSymKeyID, err := a.shh.AddSymKeyFromPassword(MailServerPassword)
	if err != nil {
		return req, err
	}

	req = shhext.MessagesRequest{
		MailServerPeer: enode,
		From:           uint32(params.From),  // TODO: change to int in status-go
		To:             uint32(params.To),    // TODO: change to int in status-go
		Limit:          uint32(params.Limit), // TODO: change to int in status-go
		SymKeyID:       mailSymKeyID,
	}

	if params.IsPublic() {
		topic, err := PublicChatTopic(params.ChatName)
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	} else {
		topic, err := PrivateChatTopic()
		if err != nil {
			return req, err
		}
		req.Topics = append(req.Topics, topic)
	}

	return
}

// whisperSubscription encapsulates a Whisper filter.
type whisperSubscription struct {
	shh      *whisper.Whisper
	filterID string
}

// newWhisperSubscription returns a new whisperSubscription.
func newWhisperSubscription(shh *whisper.Whisper, filterID string) *whisperSubscription {
	return &whisperSubscription{shh, filterID}
}

// Messages retrieves a list of messages for a given filter.
func (s whisperSubscription) Messages() ([]*protocol.Message, error) {
	f := s.shh.GetFilter(s.filterID)
	if f == nil {
		return nil, errors.New("filter does not exist")
	}

	items := f.Retrieve()
	result := make([]*protocol.Message, 0, len(items))

	for _, item := range items {
		decoded, err := protocol.DecodeMessage(item.Payload)
		if err != nil {
			log.Printf("failed to decode message: %v", err)
			continue
		}

		result = append(result, &protocol.Message{
			Decoded:   decoded,
			Hash:      item.EnvelopeHash.Bytes(),
			SigPubKey: item.SigToPubKey(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Decoded.Clock < result[j].Decoded.Clock
	})

	return result, nil
}

// Unsubscribe removes the subscription.
func (s whisperSubscription) Unsubscribe() error {
	return s.shh.Unsubscribe(s.filterID)
}
