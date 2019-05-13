package adapters

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/status-im/mvds"
	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
)

// DataSyncClient is an adapter for MVDS
// that implements the Protocol interface.
type DataSyncClient struct {
	sync mvds.Node
	t    DataSyncWhisperTransport
}

// Subscribe subscribes to a public chat using the Whisper service.
func (c *DataSyncClient) Subscribe(ctx context.Context, messages chan<- *protocol.Message, options protocol.SubscribeOptions) (*protocol.Subscription, error) {
	return c.t.subscribe(messages, options)
}

// Send appends a message to the data sync node for later sending.
func (c *DataSyncClient) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	topic, err := topic(options)
	if err != nil {
		return nil, err
	}

	id, err := c.sync.AppendMessage(toGroupId(topic), data)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

func (*DataSyncClient) Request(ctx context.Context, params protocol.RequestOptions) error {
	panic("implement me")
}

type DataSyncWhisperTransport struct {
	shh         *whisper.Whisper
	keysManager *whisperClientKeysManager

	packets chan mvds.Packet
}

func (t *DataSyncWhisperTransport) Watch() mvds.Packet {
	return <-t.packets
}

// Send sends a new message using the Whisper service.
func (t *DataSyncWhisperTransport) Send(group mvds.GroupID, _ mvds.PeerId, peer mvds.PeerId, payload mvds.Payload) error {
	data, err := proto.Marshal(&payload)

	newMessage, err := newNewMessage(t.keysManager, data)
	if err != nil {
		return err
	}

	newMessage.Topic = toTopicType(group)

	// @todo set SymKeyID or PublicKey depending on chat type

	// we are only assuming private chats
	k := ecdsa.PublicKey(peer)
	newMessage.PublicKey = crypto.FromECDSAPub(&k)

	_, err = whisper.NewPublicWhisperAPI(t.shh).Post(context.Background(), newMessage.ToWhisper())
	return err
}

func (t *DataSyncWhisperTransport) subscribe(in chan<- *protocol.Message, options protocol.SubscribeOptions) (*protocol.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter := newFilter(t.keysManager)
	if err := updateFilterFromSubscribeOptions(filter, options); err != nil {
		return nil, err
	}

	filterID, err := t.shh.Subscribe(filter.ToWhisper())
	if err != nil {
		return nil, err
	}

	subWhisper := newWhisperSubscription(t.shh, filterID)
	sub := protocol.NewSubscription()

	go func() {
		defer subWhisper.Unsubscribe() // nolint: errcheck

		tick := time.NewTicker(time.Second)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				received, err := subWhisper.Messages()
				if err != nil {
					sub.Cancel(err)
					return
				}

				for _, item := range received {
					payload := t.handlePayload(item)

					t.packets <- mvds.Packet{
						Group:   toGroupId(item.Topic),
						Sender:  mvds.PeerId(*item.Src),
						Payload: *payload,
					}

					messages := t.decodeMessages(*payload)
					for _, m := range messages {
						m.SigPubKey = item.Src
						in <- m
					}
				}
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}
// @todo return error?
func (t *DataSyncWhisperTransport) handlePayload(received *whisper.ReceivedMessage) *mvds.Payload {
	payload := &mvds.Payload{}
	err := proto.Unmarshal(received.Payload, payload)
	if err != nil {
		log.Printf("failed to decode message %#+x: %v", received.EnvelopeHash.Bytes(), err)
		return nil // @todo
	}

	return payload
}


func (t *DataSyncWhisperTransport) decodeMessages(payload mvds.Payload) []*protocol.Message {
	messages := make([]*protocol.Message, 0)

	for _, message := range payload.Messages {
		decoded, err := protocol.DecodeMessage(message.Body)
		if err != nil {
			// @todo log or something?
			continue
		}

		decoded.ID = messageID(*message)

		messages = append(messages, &decoded)
	}

	return messages
}

func toGroupId(topicType whisper.TopicType) mvds.GroupID {
	g := mvds.GroupID{}
	copy(g[:], topicType[:])
	return g
}

func toTopicType(g mvds.GroupID) whisper.TopicType {
	t := whisper.TopicType{}
	copy(t[:], g[:4])
	return t
}

func messageID(m mvds.Message) []byte {
	t := make([]byte, 8)
	binary.LittleEndian.PutUint64(t, uint64(m.Timestamp))

	b := append([]byte("MESSAGE_ID"), m.GroupId[:]...)
	b = append(b, t...)
	b = append(b, m.Body...)

	r := sha256.Sum256(b)
	hash := make([]byte, len(r))
	copy(hash[:], r[:])

	return hash
}
