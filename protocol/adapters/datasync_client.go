package adapters

import (
	"context"
	"crypto/ecdsa"
	"log"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/status-im/mvds/node"
	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	"github.com/status-im/mvds/transport"
	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
)

// DataSyncClient is an adapter for MVDS
// that implements the Protocol interface.
type DataSyncClient struct {
	sync *node.Node
	t    *DataSyncWhisperTransport
}

func NewDataSyncClient(sync *node.Node, t *DataSyncWhisperTransport) *DataSyncClient {
	go sync.Start()

	return &DataSyncClient{
		sync: sync,
		t:    t,
	}
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

	if options.ChatName == "" {
		return nil, errors.New("missing chat name")
	}

	topic, err := ToTopic(options.ChatName)
	if err != nil {
		return nil, err
	}

	gid := toGroupId(topic)

	if options.Recipient != nil {
		c.Peer(gid, options.Recipient)
	}

	// @todo store somewhere that messages for gid == public

	// @todo Peer for m.keys.AddOrGetSymKeyFromPassword(name) to simulate a Peer

	id, err := c.sync.AppendMessage(gid, data)
	if err != nil {
		return nil, err
	}

	return id[:], nil
}

func (*DataSyncClient) Request(ctx context.Context, params protocol.RequestOptions) error {
	return nil
}

func (c *DataSyncClient) Peer(id state.GroupID, peer *ecdsa.PublicKey) {
	if peer == nil {
		return
	}

	p := PublicKeyToPeerID(*peer)

	if c.sync.IsPeerInGroup(id, p) {
		return
	}

	c.sync.AddPeer(id, p)
}

type DataSyncWhisperTransport struct {
	shh         *whisper.Whisper
	keysManager *whisperServiceKeysManager

	packets chan transport.Packet
}

func NewDataSyncWhisperTransport(shh *whisper.Whisper, privateKey *ecdsa.PrivateKey) *DataSyncWhisperTransport {
	return &DataSyncWhisperTransport{
		shh: shh,
		keysManager: &whisperServiceKeysManager{
			shh:               shh,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
		packets: make(chan transport.Packet),
	}
}

func (t *DataSyncWhisperTransport) Watch() transport.Packet {
	return <-t.packets
}

// Send sends a new message using the Whisper service.
func (t *DataSyncWhisperTransport) Send(group state.GroupID, _ state.PeerID, peer state.PeerID, payload protobuf.Payload) error {
	data, err := proto.Marshal(&payload)
	if err != nil {
		return err
	}

	newMessage, err := newNewMessage(t.keysManager, data)
	if err != nil {
		return err
	}

	newMessage.Topic = toTopicType(group)

	// @todo set SymKeyID or PublicKey depending on chat type
	newMessage.PublicKey = peer[:]

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
					payload, err := t.handlePayload(item)
					if err != nil {
						log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
						continue
					}

					t.packets <- transport.Packet{
						Group:   toGroupId(item.Topic),
						Sender:  PublicKeyToPeerID(*item.Src),
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

func (t DataSyncWhisperTransport) handlePayload(received *whisper.ReceivedMessage) (*protobuf.Payload, error) {
	payload := &protobuf.Payload{}
	err := proto.Unmarshal(received.Payload, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (t DataSyncWhisperTransport) decodeMessages(payload protobuf.Payload) []*protocol.Message {
	messages := make([]*protocol.Message, 0)

	for _, message := range payload.Messages {
		decoded, err := protocol.DecodeMessage(message.Body)
		if err != nil {
			// @todo log or something?
			continue
		}

		id := state.ID(*message)
		decoded.ID = id[:]

		messages = append(messages, &decoded)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Clock < messages[j].Clock
	})

	return messages
}

// CalculateSendTime calculates the next epoch
// at which a message should be sent.
func CalculateSendTime(count uint64, time int64) int64 {
	return time + int64(count*2) // @todo this should match that time is increased by whisper periods, aka we only retransmit the first time when a message has expired.
}

func toGroupId(topicType whisper.TopicType) state.GroupID {
	g := state.GroupID{}
	copy(g[:], topicType[:])
	return g
}

func toTopicType(g state.GroupID) whisper.TopicType {
	t := whisper.TopicType{}
	copy(t[:], g[:4])
	return t
}

func PublicKeyToPeerID(k ecdsa.PublicKey) state.PeerID {
	var p state.PeerID
	copy(p[:], crypto.FromECDSAPub(&k))
	return p
}
