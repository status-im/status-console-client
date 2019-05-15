package client

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/stretchr/testify/require"
)

func TestContactReplacedBySameName(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	contact := Contact{
		Name:      "first",
		Type:      ContactPublicRoom,
		PublicKey: &pk.PublicKey,
		Topic:     "first",
	}
	require.NoError(t, db.SaveContacts([]Contact{contact}))
	require.NoError(t, db.SaveContacts([]Contact{contact}))
	rst, err := db.Contacts()
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, contact.Name, rst[0].Name)
	require.Equal(t, contact.Type, rst[0].Type)
	require.Equal(t, contact.PublicKey.X, rst[0].PublicKey.X)
	require.Equal(t, contact.PublicKey.Y, rst[0].PublicKey.Y)
}

func TestMessagesFilteredAndOrderedByTimestamp(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	contact := Contact{
		Name:      "test",
		Type:      ContactPublicRoom,
		PublicKey: &pk.PublicKey,
		Topic:     "first",
	}
	require.NoError(t, db.SaveContacts([]Contact{contact}))
	contacts, err := db.Contacts()
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	msg1 := protocol.Message{
		ID:        []byte("hello1"),
		SigPubKey: &pk.PublicKey,
		Timestamp: 10000,
	}
	msg2 := protocol.Message{
		ID:        []byte("hello2"),
		SigPubKey: &pk.PublicKey,
		Timestamp: 4000,
	}
	msg3 := protocol.Message{
		ID:        []byte("hello3"),
		SigPubKey: &pk.PublicKey,
		Timestamp: 2000,
	}

	last, err := db.SaveMessages(contact, []*protocol.Message{&msg3, &msg1, &msg2})
	require.NoError(t, err)
	require.Equal(t, int64(3), last)
	msgs, err := db.Messages(contact, time.Unix(3, 0), time.Unix(11, 0))
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, msg2.Timestamp, msgs[0].Timestamp)
	require.Equal(t, msg1.Timestamp, msgs[1].Timestamp)
}

func TestSaveMessagesUniqueConstraint(t *testing.T) {
	contact := Contact{
		Name:  "test",
		Type:  ContactPublicRoom,
		Topic: "first",
	}
	sameid := []byte("1")
	msg1 := protocol.Message{
		ID: sameid,
	}
	msg2 := protocol.Message{
		ID: sameid,
	}
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()

	_, err = db.SaveMessages(contact, []*protocol.Message{&msg1, &msg2})
	require.EqualError(t, err, ErrMsgAlreadyExist.Error())
}

func TestGetLastMessageClock(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	count := 10
	messages := make([]*protocol.Message, count)
	for i := range messages {
		// set clock in reverse order to prevent simply selecting last message from table
		messages[i] = &protocol.Message{
			ID:    []byte{byte(i)},
			Clock: int64(count - i),
		}
	}
	contact := Contact{
		Name:  "test",
		Type:  ContactPublicRoom,
		Topic: "first",
	}
	_, err = db.SaveMessages(contact, messages)
	require.NoError(t, err)
	last, err := db.LastMessageClock(contact)
	require.NoError(t, err)
	require.Equal(t, int64(count), last)
}

func TestPublicContactExist(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	contact := Contact{
		Name:      "first",
		Type:      ContactPublicKey,
		PublicKey: &pk.PublicKey,
		Topic:     "first",
	}
	require.NoError(t, db.SaveContacts([]Contact{contact}))
	exists, err := db.PublicContactExist(contact)
	require.NoError(t, err)
	require.True(t, exists, "contact expected to exist in database")
}

func BenchmarkLoadMessages(b *testing.B) {
	db, err := InitializeTmpDB()
	require.NoError(b, err)
	defer db.Close()
	pk, err := crypto.GenerateKey()
	require.NoError(b, err)
	contacts := []Contact{
		{
			Name:      "first",
			Type:      ContactPublicKey,
			PublicKey: &pk.PublicKey,
			Topic:     "test",
		},
		{
			Name:      "second",
			Type:      ContactPublicKey,
			PublicKey: &pk.PublicKey,
			Topic:     "test",
		},
		{
			Name:      "third",
			Type:      ContactPublicKey,
			PublicKey: &pk.PublicKey,
			Topic:     "test",
		},
	}
	count := 10000
	require.NoError(b, db.SaveContacts(contacts))
	for j, c := range contacts {
		messages := make([]*protocol.Message, count)
		for i := range messages {
			id := [8]byte{}
			id[0] = byte(j)
			binary.PutVarint(id[1:], int64(i))
			messages[i] = &protocol.Message{
				SigPubKey: c.PublicKey,
				ID:        id[:],
			}
		}
		_, err = db.SaveMessages(c, messages)
		require.NoError(b, err)

	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rst, err := db.GetNewMessages(contacts[0], 0)
		require.NoError(b, err)
		require.Len(b, rst, count)
	}
}
