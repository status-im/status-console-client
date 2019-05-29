package client

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/stretchr/testify/require"
)

func TestStreamHandlerForContact(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()

	contact := Contact{Name: "test", Type: ContactPublicRoom}
	handler := StreamHandlerForContact(contact, db)
	msg := protocol.Message{
		ID: []byte{1},
	}

	require.NoError(t, handler(&msg))

	msgs, err := db.GetNewMessages(contact, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, msg.ID, msgs[0].ID)
}

func TestPrivateStreamSavesNewContactsAndMessages(t *testing.T) {
	db, err := InitializeTmpDB()
	require.NoError(t, err)
	defer db.Close()
	pkey, err := crypto.GenerateKey()
	require.NoError(t, err)
	handler := StreamHandlerMultiplexed(db)
	msg := protocol.Message{
		ID:        []byte{1},
		SigPubKey: &pkey.PublicKey,
	}

	require.NoError(t, handler(&msg))

	// assert a new contact with proper state
	contacts, err := db.Contacts()
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.Equal(t, &pkey.PublicKey, contacts[0].PublicKey)
	require.Equal(t, ContactNew, contacts[0].State)

	// aassert saved messages
	msgs, err := db.GetNewMessages(contacts[0], 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, &pkey.PublicKey, msgs[0].SigPubKey)
}
