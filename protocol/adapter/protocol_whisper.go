package adapter

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"time"

	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"

	msgfilter "github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/messaging/multidevice"
	"github.com/status-im/status-go/messaging/publisher"
)

type ProtocolWhisperAdapter struct {
	c         Config
	transport transport.WhisperTransport
	publisher *publisher.Publisher
	messages  chan *protocol.ReceivedMessages
}

// ProtocolWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*ProtocolWhisperAdapter)(nil)

type Config struct {
	PFSEnabled bool
}

func NewProtocolWhisperAdapter(t transport.WhisperTransport, p *publisher.Publisher, c Config) *ProtocolWhisperAdapter {
	return &ProtocolWhisperAdapter{
		c:         c,
		transport: t,
		publisher: p,
		messages:  make(chan *protocol.ReceivedMessages),
	}
}

func (w *ProtocolWhisperAdapter) GetMessagesChan() chan *protocol.ReceivedMessages {
	return w.messages
}

// Subscribe listens to new messages.
func (w *ProtocolWhisperAdapter) Subscribe(
	ctx context.Context,
	messages chan *protocol.StatusMessage,
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
			whisperMessage := whisper.ToWhisperMessage(item)
			message, err := w.decodeMessage(whisperMessage)
			if err != nil {
				log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
				continue
			}
			messages <- message
		}
	}()

	return sub, nil
}

func (w *ProtocolWhisperAdapter) decodeMessage(message *whisper.Message) (*protocol.StatusMessage, error) {
	payload := message.Payload
	var err error
	var publicKey *ecdsa.PublicKey
	if publicKey, err = crypto.UnmarshalPubkey(message.Sig); err != nil {
		return nil, errors.New("can't unmarshal pubkey")
	}
	hash := message.Hash

	if w.publisher != nil {
		if err := w.publisher.ProcessMessage(message, hash); err != nil {
			log.Printf("failed to process message: %#x: %v", hash, err)
		}
		payload = message.Payload
	}

	decoded, err := protocol.DecodeMessage(payload)
	if err != nil {
		return nil, err
	}
	decoded.ID = hash
	decoded.SigPubKey = publicKey

	return decoded, nil
}

func (w *ProtocolWhisperAdapter) OnNewMessages(messages []*msgfilter.Messages) {
	for _, filterMessages := range messages {

		receivedMessages := &protocol.ReceivedMessages{}

		var options protocol.ChatOptions

		if filterMessages.Chat.OneToOne || filterMessages.Chat.Negotiated || filterMessages.Chat.Discovery {
			options.ChatName = filterMessages.Chat.Identity
			options.OneToOne = true
		} else {
			options.ChatName = filterMessages.Chat.ChatID
		}

		receivedMessages.ChatOptions = options
		for _, message := range filterMessages.Messages {
			decodedMessage, err := w.decodeMessage(message)
			if err != nil {
				log.Printf("failed to process message: %v", err)
				continue
			}
			receivedMessages.Messages = append(receivedMessages.Messages, decodedMessage)

		}

		w.messages <- receivedMessages
	}
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (w *ProtocolWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	var newMessage *whisper.NewMessage

	if w.c.PFSEnabled {
		privateKey := w.transport.KeysManager().PrivateKey()

		var err error

		// TODO: rethink this
		if options.Recipient != nil {
			newMessage, err = w.publisher.CreateDirectMessage(privateKey, options.Recipient, false, data)
		} else {
			// Public messages are not wrapped (i.e have not bundle),
			// when sending in public chats as it would be a breaking change.
			// When we send a contact code, we send a public message but wrapped,
			// as PFS enabled client are the only ones using it.
			// Thus, we keep it to false here.
			newMessage, err = w.publisher.CreatePublicMessage(privateKey, options.ChatName, data, false)
		}

		if err != nil {
			return nil, err
		}
	} else {
		message, err := NewNewMessage(w.transport.KeysManager(), data)
		if err != nil {
			return nil, err
		}
		if err := updateNewMessageFromSendOptions(message, options); err != nil {
			return nil, err
		}

		newMessage = &message.NewMessage
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

func chatOptionsToFilterChat(chatOption protocol.ChatOptions) *msgfilter.Chat {
	if chatOption.Recipient != nil {
		identityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(chatOption.Recipient))
		return &msgfilter.Chat{
			ChatID:   fmt.Sprintf("0x%s", identityStr),
			OneToOne: true,
			Identity: identityStr,
		}
	}

	// Public chat
	return &msgfilter.Chat{
		ChatID: chatOption.ChatName,
	}
}

func (w *ProtocolWhisperAdapter) SetInstallationMetadata(ctx context.Context, installationID string, data *multidevice.InstallationMetadata) error {
	return w.publisher.SetInstallationMetadata(installationID, data)
}

func (w *ProtocolWhisperAdapter) LoadChats(ctx context.Context, params []protocol.ChatOptions) error {
	var filterChats []*msgfilter.Chat
	var err error
	for _, chatOption := range params {
		filterChats = append(filterChats, chatOptionsToFilterChat(chatOption))
	}
	_, err = w.publisher.LoadFilters(filterChats)
	if err != nil {
		return err
	}
	return nil
}

func (w *ProtocolWhisperAdapter) RemoveChats(ctx context.Context, params []protocol.ChatOptions) error {
	var filterChats []*msgfilter.Chat
	for _, chatOption := range params {
		filterChat := chatOptionsToFilterChat(chatOption)
		// We only remove public chats, as we can't remove one-to-one
		// filters as otherwise we won't be receiving any messages
		// from the user, which is the equivalent of a block feature
		if !filterChat.OneToOne {
			filterChats = append(filterChats, filterChat)
		}

	}

	return w.publisher.RemoveFilters(filterChats)
}
