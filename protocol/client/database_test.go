package client

import (
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

	require.NoError(t, db.SaveMessages(contact, []*protocol.Message{&msg3, &msg1, &msg2}))
	msgs, err := db.Messages(contact, time.Unix(3, 0), time.Unix(11, 0))
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, msg2.Timestamp, msgs[0].Timestamp)
	require.Equal(t, msg1.Timestamp, msgs[1].Timestamp)
}
