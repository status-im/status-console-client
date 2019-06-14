package datasync

import (
	"context"

	"github.com/gogo/protobuf/proto"

	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	"github.com/status-im/mvds/transport"

	"github.com/status-im/status-console-client/protocol/adapter"
	prototrns "github.com/status-im/status-console-client/protocol/transport"

	whisper "github.com/status-im/whisper/whisperv6"
)

type DataSyncNodeTransport struct {
	transport prototrns.WhisperTransport
	packets   chan transport.Packet
}

func NewDataSyncNodeTransport(t prototrns.WhisperTransport) *DataSyncNodeTransport {
	return &DataSyncNodeTransport{
		transport: t,
		packets:   make(chan transport.Packet),
	}
}

func (t *DataSyncNodeTransport) AddPacket(p transport.Packet) {
	t.packets <- p
}

func (t *DataSyncNodeTransport) Watch() transport.Packet {
	return <-t.packets
}

func (t *DataSyncNodeTransport) Send(group state.GroupID, _ state.PeerID, peer state.PeerID, payload protobuf.Payload) error {
	data, err := proto.Marshal(&payload)
	if err != nil {
		return err
	}

	newMessage, err := adapter.NewNewMessage(t.transport.KeysManager(), data)
	if err != nil {
		return err
	}

	newMessage.Topic = toTopicType(group)

	// @todo set SymKeyID or PublicKey depending on chat type
	newMessage.PublicKey = peer[:]

	_, err = t.transport.Send(context.Background(), newMessage.NewMessage)
	return err
}

func toTopicType(g state.GroupID) whisper.TopicType {
	t := whisper.TopicType{}
	copy(t[:], g[:4])
	return t
}

// CalculateSendTime calculates the next epoch
// at which a message should be sent.
func CalculateSendTime(count uint64, time int64) int64 {
	return time + int64(count*2) // @todo this should match that time is increased by whisper periods, aka we only retransmit the first time when a message has expired.
}
