package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/gob"
	"io"
	"log"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/status-im/status-console-client/protocol/v1"
)

func init() {
	gob.Register(&secp256k1.BitCurve{})
}

type Database struct {
	db *leveldb.DB
}

func NewDatabase(path string) (*Database, error) {
	if path == "" {
		return &Database{}, nil
	}

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) Messages(c Contact, from, to int64) (result []*protocol.ReceivedMessage, err error) {
	start := d.keyFromContact(c, from, nil)
	limit := d.keyFromContact(c, to+1, nil) // because iter is right-exclusive

	iter := d.db.NewIterator(&util.Range{Start: start, Limit: limit}, nil)
	for iter.Next() {
		value := iter.Value()
		buf := bytes.NewBuffer(value)
		dec := gob.NewDecoder(buf)

		var m protocol.ReceivedMessage

		err = dec.Decode(&m)
		if err != nil {
			return
		}

		result = append(result, &m)
	}

	iter.Release()
	err = iter.Error()

	return
}

func (d *Database) SaveMessages(c Contact, messages ...*protocol.ReceivedMessage) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	batch := new(leveldb.Batch)
	for _, m := range messages {
		// incoming Timestamp is in ms
		key := d.keyFromContact(c, m.Decoded.Timestamp/1000, m.Hash)

		log.Printf("saving messages with key: %x", key)

		if err := enc.Encode(m); err != nil {
			return err
		}

		data := buf.Bytes()
		value := make([]byte, len(data))
		copy(value, data)
		// The read value needs to be copied to another slice
		// because a slice returned by Bytes() is valid only until
		// another write.
		// As we batch writes and wait untill the loop is finished,
		// slices must be available later.
		batch.Put(key, value)
		buf.Reset()
	}

	return d.db.Write(batch, nil)
}

var contactsListKey = []byte("contacts-list")

func (d *Database) Contacts() ([]Contact, error) {
	value, err := d.db.Get(contactsListKey, nil)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(value)
	dec := gob.NewDecoder(buf)

	var contacts []Contact

	if err := dec.Decode(&contacts); err != nil {
		return nil, err
	}

	return contacts, nil
}

func (d *Database) SaveContacts(contacts []Contact) error {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(contacts); err != nil {
		return err
	}

	return d.db.Put([]byte(contactsListKey), buf.Bytes(), nil)
}

func (d *Database) prefixFromContact(c Contact) []byte {
	h := sha1.New()
	io.WriteString(h, c.String())
	return h.Sum(nil)
}

func (d *Database) keyFromContact(c Contact, t int64, hash []byte) []byte {
	var key [27 + 32]byte // TODO: recalculate this

	copy(key[:], d.prefixFromContact(c))
	binary.BigEndian.PutUint64(key[20:], uint64(t))

	if hash != nil {
		copy(key[28:], hash)
	}

	return key[:]
}