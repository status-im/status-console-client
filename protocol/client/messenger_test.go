package client

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/status-im/status-console-client/protocol/subscription"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/stretchr/testify/require"
)

type requestsMock struct {
	requests []protocol.RequestOptions
}

func (proto *requestsMock) Send(ctx context.Context, data []byte, options protocol.SendOptions) ([]byte, error) {
	return nil, nil
}

func (proto *requestsMock) Subscribe(ctx context.Context, messages chan<- *protocol.Message, options protocol.SubscribeOptions) (*subscription.Subscription, error) {
	return nil, nil
}

func (proto *requestsMock) Request(ctx context.Context, params protocol.RequestOptions) error {
	proto.requests = append(proto.requests, params)
	return nil
}

func TestRequestHistoryOneRequest(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	proto := &requestsMock{}
	m := NewMessenger(nil, proto, db)

	require.NoError(t, db.SaveContacts([]Contact{
		{Name: "first", Type: ContactPublicRoom},
		{Name: "second", Type: ContactPublicRoom}}))
	require.NoError(t, m.RequestAll(context.TODO(), true))

	require.Len(t, proto.requests, 1)
	histories, err := db.Histories()
	require.NoError(t, err)
	require.Len(t, histories, 2)
	require.Equal(t, histories[0].Synced, proto.requests[0].To)
	require.Equal(t, histories[1].Synced, proto.requests[0].To)
}

func TestRequestHistoryTwoRequest(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	proto := &requestsMock{}
	m := NewMessenger(nil, proto, db)

	contacts := []Contact{
		{Name: "first", Type: ContactPublicRoom, Topic: "first"},
		{Name: "second", Type: ContactPublicRoom, Topic: "second"},
		{Name: "third", Type: ContactPublicRoom, Topic: "third"},
	}
	require.NoError(t, db.SaveContacts(contacts))
	histories := []History{{Synced: time.Now().Add(-time.Hour).Unix(), Contact: contacts[0]}}
	require.NoError(t, db.UpdateHistories(histories))
	require.NoError(t, m.RequestAll(context.TODO(), true))

	require.Len(t, proto.requests, 2)
	sort.Slice(proto.requests, func(i, j int) bool {
		return proto.requests[i].From < proto.requests[j].From
	})
	require.Len(t, proto.requests[0].Chats, 2)
	require.Len(t, proto.requests[1].Chats, 1)
	require.Equal(t, histories[0].Contact.Name, proto.requests[1].Chats[0].ChatName)
	require.Equal(t, histories[0].Synced, proto.requests[1].From)

}
