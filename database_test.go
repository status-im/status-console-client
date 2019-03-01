package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-console-client/protocol/v1"
)

const testPublicKey = "0x0493ac727e70ea62c4428caddf4da301ca67b699577988d6a782898acfd813addf79b2a2ca2c411499f2e0a12b7de4d00574cbddb442bec85789aea36b10f46895"

func TestKeyFromContact(t *testing.T) {
	db, err := NewDatabase("")
	require.NoError(t, err)

	c, err := NewContactWithPublicKey("test-priv-contact", testPublicKey)
	require.NoError(t, err)

	now := time.Now().Unix()
	key := db.keyFromContact(c, now, nil)
	require.Len(t, key, 59)
	require.True(t, bytes.Equal(key, db.keyFromContact(c, now, nil)))
}

func TestCompareKeyFromContact(t *testing.T) {
	db, err := NewDatabase("")
	require.NoError(t, err)

	c, err := NewContactWithPublicKey("test-priv-contact", testPublicKey)
	require.NoError(t, err)

	now := time.Now().Unix()
	key1 := db.keyFromContact(c, now, []byte{0x01})
	key2 := db.keyFromContact(c, now+1, []byte{0x02})
	require.Equal(t, -1, bytes.Compare(key1, key2))
}

func TestSaveContacts(t *testing.T) {
	path, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(path)

	db, err := NewDatabase(path)
	require.NoError(t, err)

	c, err := NewContactWithPublicKey("test-priv-contact", testPublicKey)
	require.NoError(t, err)

	// save
	err = db.SaveContacts([]Contact{c})
	require.NoError(t, err)

	// retrieve
	contacts, err := db.Contacts()
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.Equal(t, c, contacts[0])
}

func TestSaveAndGetMessage(t *testing.T) {
	path, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(path)

	db, err := NewDatabase(path)
	require.NoError(t, err)

	now := time.Now().Unix()

	c := Contact{
		Name: "test-pub-chat",
		Type: ContactPublicChat,
	}
	m := protocol.ReceivedMessage{
		Decoded: protocol.StatusMessage{
			Text:      "test",
			ContentT:  "test-content-type",
			MessageT:  "test-message-type",
			Clock:     now * 1000, // TODO: unify
			Timestamp: now * 1000, // TODO: unify
		},
	}

	m1 := m
	m1.Hash = []byte{0x01}
	err = db.SaveMessages(c, &m1)
	require.NoError(t, err)

	m2 := m
	m2.Hash = []byte{0x02}
	err = db.SaveMessages(c, &m2)
	require.NoError(t, err)

	messages, err := db.Messages(c, now-1, now+1)
	require.NoError(t, err)
	require.Len(t, messages, 2)
}
