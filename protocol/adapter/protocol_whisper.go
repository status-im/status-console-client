package adapter

import (
	"context"
	"log"
	"time"

	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"

	msgfilter "github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/messaging/publisher"
)

type ProtocolWhisperAdapter struct {
	transport transport.WhisperTransport
	publisher *publisher.Publisher
}

// ProtocolWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*ProtocolWhisperAdapter)(nil)

func NewProtocolWhisperAdapter(t transport.WhisperTransport, p *publisher.Publisher) *ProtocolWhisperAdapter {
	return &ProtocolWhisperAdapter{
		transport: t,
		publisher: p,
	}
}

// Subscribe listens to new messages.
func (w *ProtocolWhisperAdapter) Subscribe(
	ctx context.Context,
	messages chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*subscription.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter := newFilter(w.transport.KeysManager())
	if err := updateFilterFromSubscribeOptions(filter, options); err != nil {
		return nil, err
	}

	// Messages income in batches and hence a buffered channel is used.
	in := make(chan *whisper.ReceivedMessage, 1024)
	sub, err := w.transport.Subscribe(ctx, in, filter.Filter)
	if err != nil {
		return nil, err
	}

	go func() {
		for item := range in {
			message, err := w.decodeMessage(item)
			if err != nil {
				log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
				continue
			}
			messages <- message
		}
	}()

	return sub, nil
}

func (w *ProtocolWhisperAdapter) decodeMessage(message *whisper.ReceivedMessage) (*protocol.Message, error) {
	payload := message.Payload
	publicKey := message.SigToPubKey()
	hash := message.EnvelopeHash.Bytes()
	msg := whisper.ToWhisperMessage(message)

	if w.publisher != nil {
		if err := w.publisher.ProcessMessage(msg, msg.Hash); err != nil {
			// TODO(adam): err can be chat.ErrNotPairedDevice, chat.ErrDeviceNotFound and in that case it requires special handling.
			log.Printf("failed to process message: %#x: %v", hash, err)
		}
		payload = msg.Payload
	}

	decoded, err := protocol.DecodeMessage(payload)
	if err != nil {
		return nil, err
	}
	decoded.ID = hash
	decoded.SigPubKey = publicKey

	return &decoded, nil
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (w *ProtocolWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	var newMessage *whisper.NewMessage

	if w.publisher != nil {
		privateKey := w.transport.KeysManager().PrivateKey()

		var err error

		// TODO: rethink this
		if options.Recipient != nil {
			newMessage, err = w.publisher.CreateDirectMessage(privateKey, options.Recipient, false, data)
		} else {
			_, filterErr := w.publisher.LoadFilter(&msgfilter.Chat{
				ChatID: options.ChatName,
			})
			if filterErr != nil {
				return nil, errors.Wrap(filterErr, "failed to load filter")
			}

			// TODO(adam): when wrap (the last argument) is important?
			newMessage, err = w.publisher.CreatePublicMessage(privateKey, options.ChatName, data, false)
		}

		if err != nil {
			return nil, err
		}
	}

	return w.transport.Send(ctx, *newMessage)
}

// Request retrieves historic messages.
func (w *ProtocolWhisperAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	transOptions := transport.RequestOptions{
		Password: MailServerPassword,
		Topics:   []whisper.TopicType{},
		From:     params.From,
		To:       params.To,
		Limit:    params.Limit,
	}
	for _, chat := range params.Chats {
		topic, err := ToTopic(chat.ChatName)
		if err != nil {
			return err
		}
		transOptions.Topics = append(transOptions.Topics, topic)
	}
	now := time.Now()
	err := w.transport.Request(ctx, transOptions)
	log.Printf("[ProtocolWhisperAdapter::Request] took %s", time.Since(now))
	return err
}
