package adapter

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"

	"github.com/gogo/protobuf/proto"

	"github.com/status-im/mvds/node"
	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	dstrns "github.com/status-im/mvds/transport"

	dspeer "github.com/status-im/status-console-client/protocol/datasync/peer"
	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/transport"
	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/ethereum/go-ethereum/crypto"
	msgfilter "github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/messaging/multidevice"
	"github.com/status-im/status-go/messaging/publisher"
	whisper "github.com/status-im/whisper/whisperv6"
)

type PacketHandler interface {
	AddPacket(dstrns.Packet)
}

type DataSyncWhisperAdapter struct {
	node      *node.Node
	transport transport.WhisperTransport
	publisher *publisher.Publisher
	packets   PacketHandler
	messages  chan *protocol.ReceivedMessages
}

// DataSyncWhisperAdapter must implement Protocol interface.
var _ protocol.Protocol = (*DataSyncWhisperAdapter)(nil)

func NewDataSyncWhisperAdapter(n *node.Node, t transport.WhisperTransport, h PacketHandler, p *publisher.Publisher) *DataSyncWhisperAdapter {
	return &DataSyncWhisperAdapter{
		node:      n,
		transport: t,
		packets:   h,
		publisher: p,
		messages:  make(chan *protocol.ReceivedMessages),
	}
}

// Subscribe listens to new messages.
func (w *DataSyncWhisperAdapter) Subscribe(
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
			payload, err := w.decodePayload(item)
			if err != nil {
				log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
				continue
			}

			packet := dstrns.Packet{
				Group:   toGroupId(item.Topic),
				Sender:  dspeer.PublicKeyToPeerID(*item.Src),
				Payload: payload,
			}
			w.packets.AddPacket(packet)

			for _, m := range w.decodeMessages(payload) {
				m.SigPubKey = item.Src
				messages <- m
			}
		}
	}()

	return sub, nil
}

func (w *DataSyncWhisperAdapter) decodePayload(message *whisper.ReceivedMessage) (payload protobuf.Payload, err error) {
	err = proto.Unmarshal(message.Payload, &payload)
	return
}

func (w *DataSyncWhisperAdapter) OnNewMessages(messages []*msgfilter.Messages) {
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

func (w *DataSyncWhisperAdapter) decodeMessage(message *whisper.Message) (*protocol.StatusMessage, error) {
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

func (w *DataSyncWhisperAdapter) decodeMessages(payload protobuf.Payload) []*protocol.StatusMessage {
	messages := make([]*protocol.StatusMessage, 0)

	for _, message := range payload.Messages {
		decoded, err := protocol.DecodeMessage(message.Body)
		if err != nil {
			// @todo log or something?
			continue
		}

		id := state.ID(*message)
		decoded.ID = id[:]

		messages = append(messages, decoded)
	}

	return messages
}

// Send sends a message to the network.
// Identity is required as the protocol requires
// all messages to be signed.
func (w *DataSyncWhisperAdapter) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if options.ChatName == "" {
		return nil, errors.New("missing chat name")
	}

	topic, err := ToTopic(options.ChatName)
	if err != nil {
		return nil, err
	}

	gid := toGroupId(topic)

	w.peer(gid, options.Recipient)

	id, err := w.node.AppendMessage(gid, data)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

// Request retrieves historic messages.
func (w *DataSyncWhisperAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}

func (w *DataSyncWhisperAdapter) peer(id state.GroupID, peer *ecdsa.PublicKey) {
	if peer == nil {
		return
	}

	p := dspeer.PublicKeyToPeerID(*peer)

	if w.node.IsPeerInGroup(id, p) {
		return
	}

	w.node.AddPeer(id, p)
}

func toGroupId(topicType whisper.TopicType) state.GroupID {
	g := state.GroupID{}
	copy(g[:], topicType[:])
	return g
}

func (w *DataSyncWhisperAdapter) GetMessagesChan() chan *protocol.ReceivedMessages {
	return w.messages
}

func (w *DataSyncWhisperAdapter) SetInstallationMetadata(ctx context.Context, installationID string, data *multidevice.InstallationMetadata) error {
	return w.publisher.SetInstallationMetadata(installationID, data)
}

func (w *DataSyncWhisperAdapter) LoadChats(ctx context.Context, params []protocol.ChatOptions) error {
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

func (w *DataSyncWhisperAdapter) RemoveChats(ctx context.Context, params []protocol.ChatOptions) error {
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
