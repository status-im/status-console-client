package client

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/stretchr/testify/require"
)

func TestContactUniqueConstraint(t *testing.T) {
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
	require.EqualError(t, db.SaveContacts([]Contact{contact}), "UNIQUE constraint failed: user_contacts.id")
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

func TestUnreadMessages(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	contact := Contact{
		Name:  "test",
		Type:  ContactPublicRoom,
		Topic: "first",
	}
	// insert some messages
	var messages []*protocol.Message
	for i := 0; i < 4; i++ {
		var flags protocol.Flags
		if i%2 == 0 {
			// even messages are marked as read
			flags.Set(protocol.MessageRead)
		}
		m := protocol.Message{
			ID:        []byte{byte(i)},
			Timestamp: protocol.TimestampInMs(i + 1),
			Clock:     int64(i + 1),
			Flags:     flags,
		}
		messages = append(messages, &m)
	}
	_, err = db.SaveMessages(contact, messages)
	require.NoError(t, err)

	// verify that we get only unread messages
	unread, err := db.UnreadMessages(contact)
	require.NoError(t, err)
	require.Len(t, unread, 2)
	for _, m := range unread {
		require.False(t, m.Flags.Has(protocol.MessageRead))
	}
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

func TestGetOneToOneChat(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	expectedContact := Contact{
		Name:      "first",
		Type:      ContactPrivate,
		PublicKey: &pk.PublicKey,
		Topic:     "first",
	}
	require.NoError(t, db.SaveContacts([]Contact{expectedContact}))
	contact, err := db.GetOneToOneChat(&pk.PublicKey)
	require.NoError(t, err)
	require.Equal(t, &expectedContact, contact, "contact expected to exist in database")
}

func TestLoadHistories(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	c1 := Contact{
		Name: "first",
		Type: ContactPublicRoom,
	}
	c2 := Contact{
		Name: "second",
		Type: ContactPublicRoom,
	}
	require.NoError(t, db.SaveContacts([]Contact{c1, c2}))
	histories, err := db.Histories()
	require.NoError(t, err)
	require.Len(t, histories, 2)
}

func TestUpdateHistories(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	c1 := Contact{
		Name: "first",
		Type: ContactPublicRoom,
	}
	require.NoError(t, db.SaveContacts([]Contact{c1}))
	h := History{
		Synced:  100,
		Contact: c1,
	}
	require.NoError(t, db.UpdateHistories([]History{h}))
	histories, err := db.Histories()
	require.NoError(t, err)
	require.Len(t, histories, 1)
	require.Equal(t, h.Synced, histories[0].Synced)
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
			Type:      ContactPrivate,
			PublicKey: &pk.PublicKey,
			Topic:     "test",
		},
		{
			Name:      "second",
			Type:      ContactPrivate,
			PublicKey: &pk.PublicKey,
			Topic:     "test",
		},
		{
			Name:      "third",
			Type:      ContactPrivate,
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
		rst, err := db.NewMessages(contacts[0], 0)
		require.NoError(b, err)
		require.Len(b, rst, count)
	}
}
