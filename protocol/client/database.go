package client

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/gob"
	"io"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/status-im/status-console-client/protocol/v1"
)

var (
	contactsListKey = []byte("contacts-list")
)

func init() {
	gob.Register(&secp256k1.BitCurve{})
}

// Database is a wrapped around leveldb to provide storage
// for messenger data.
type Database struct {
	db *leveldb.DB
}

// NewDatabase returns a new database creating files
// in a given path directory.
func NewDatabase(path string) (*Database, error) {
	if path == "" {
		// If path is not give, use in-memory storage.
		storage := storage.NewMemStorage()
		db, err := leveldb.Open(storage, nil)
		if err != nil {
			return nil, err
		}
		return &Database{db: db}, nil
	}

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

// Messages returns all messages for a given contact
// and between from and to timestamps.
func (d *Database) Messages(c Contact, from, to int64) (result []*protocol.Message, err error) {
	start := d.keyFromContact(c, from, nil)
	limit := d.keyFromContact(c, to+1, nil) // because iter is right-exclusive

	iter := d.db.NewIterator(&util.Range{Start: start, Limit: limit}, nil)
	for iter.Next() {
		value := iter.Value()
		buf := bytes.NewBuffer(value)
		dec := gob.NewDecoder(buf)

		var m protocol.Message

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

// SaveMessages stores messages on a disk.
func (d *Database) SaveMessages(c Contact, messages []*protocol.Message) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	batch := new(leveldb.Batch)
	for _, m := range messages {
		// TODO(adam): incoming Timestamp is in ms
		key := d.keyFromContact(c, m.Decoded.Timestamp/1000, m.Hash)

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

// Contacts retrieves all saved contacts.
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

// SaveContacts saves all contacts on a disk.
func (d *Database) SaveContacts(contacts []Contact) error {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(contacts); err != nil {
		return err
	}

	return d.db.Put([]byte(contactsListKey), buf.Bytes(), nil)
}

const (
	contactPrefixLength  = sha1.Size
	timeLength           = 8
	hashLength           = 32
	keyFromContactLength = contactPrefixLength + timeLength + hashLength
)

func (d *Database) prefixFromContact(c Contact) []byte {
	h := sha1.New()
	_, _ = io.WriteString(h, c.Name)
	_, _ = io.WriteString(h, ":")
	_, _ = io.WriteString(h, strconv.Itoa(int(c.Type)))
	return h.Sum(nil)
}

func (d *Database) keyFromContact(c Contact, t int64, hash []byte) []byte {
	var key [keyFromContactLength]byte

	copy(key[:], d.prefixFromContact(c))
	binary.BigEndian.PutUint64(key[contactPrefixLength:], uint64(t))

	if hash != nil {
		copy(key[contactPrefixLength+timeLength:], hash)
	}

	return key[:]
}
