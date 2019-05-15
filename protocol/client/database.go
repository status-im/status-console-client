package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
	"github.com/status-im/migrate"
	"github.com/status-im/migrate/database/sqlcipher"
	bindata "github.com/status-im/migrate/source/go_bindata"
	"github.com/status-im/status-console-client/protocol/client/migrations"
	"github.com/status-im/status-console-client/protocol/client/sqlite"
	"github.com/status-im/status-console-client/protocol/v1"
)

func init() {
	// this is used for marshalling public key with a curve.
	gob.Register(&secp256k1.BitCurve{})
}

const (
	uniqueIDContstraint = "UNIQUE constraint failed: user_messages.id"
)

var (
	// ErrMsgAlreadyExist returned if msg already exist.
	ErrMsgAlreadyExist = errors.New("message with given ID already exist")
)

// Database is an interface for all db operations.
type Database interface {
	Close() error
	Messages(c Contact, from, to time.Time) (result []*protocol.Message, err error)
	GetNewMessages(Contact, int64) ([]*protocol.Message, error)
	SaveMessages(c Contact, messages []*protocol.Message) (int64, error)
	LastMessageClock(Contact) (int64, error)
	Contacts() ([]Contact, error)
	SaveContacts(contacts []Contact) error
	DeleteContact(Contact) error
	PublicContactExist(Contact) (bool, error)
}

// Migrate applies migrations.
func Migrate(db *sql.DB) error {
	resources := bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// InitializeTmpDB creates database in temporary directory with a random key.
// Used for tests.
func InitializeTmpDB() (TmpDatabase, error) {
	tmpfile, err := ioutil.TempFile("", "client-tests-")
	if err != nil {
		return TmpDatabase{}, err
	}
	pass := make([]byte, 4)
	_, err = rand.Read(pass)
	if err != nil {
		return TmpDatabase{}, err
	}
	db, err := InitializeDB(tmpfile.Name(), string(pass))
	if err != nil {
		return TmpDatabase{}, err
	}
	return TmpDatabase{
		SQLLiteDatabase: db,
		file:            tmpfile,
	}, nil
}

// TmpDatabase wraps SQLLiteDatabase and removes temporary file after db was closed.
type TmpDatabase struct {
	SQLLiteDatabase
	file *os.File
}

// Close closes sqlite database and removes temporary file.
func (db TmpDatabase) Close() error {
	_ = db.SQLLiteDatabase.Close()
	return os.Remove(db.file.Name())
}

// InitializeDB opens encrypted sqlite database from provided path and applies migrations.
func InitializeDB(path, key string) (SQLLiteDatabase, error) {
	db, err := sqlite.OpenDB(path, key)
	if err != nil {
		return SQLLiteDatabase{}, err
	}
	err = Migrate(db)
	if err != nil {
		return SQLLiteDatabase{}, err
	}
	return SQLLiteDatabase{db: db}, nil
}

// SQLLiteDatabase wrapper around sql db with operations common for a client.
type SQLLiteDatabase struct {
	db *sql.DB
}

// Close closes internal sqlite database.
func (db SQLLiteDatabase) Close() error {
	return db.db.Close()
}

// SaveContacts inserts or replaces provided contacts.
// TODO should it delete all previous contacts?
func (db SQLLiteDatabase) SaveContacts(contacts []Contact) (err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	stmt, err = tx.Prepare("INSERT OR REPLACE INTO user_contacts(id, name, type, state, topic, public_key) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()
	var (
		buf bytes.Buffer
	)
	for i := range contacts {
		enc := gob.NewEncoder(&buf)
		if contacts[i].PublicKey != nil {
			err = enc.Encode(contacts[i].PublicKey)
			if err != nil {
				return err
			}
		}
		pkey := append([]byte{}, buf.Bytes()...)
		buf.Reset()
		id := fmt.Sprintf("%s:%d", contacts[i].Name, contacts[i].Type)
		_, err = stmt.Exec(id, contacts[i].Name, contacts[i].Type, contacts[i].State, contacts[i].Topic, pkey)
		if err != nil {
			return err
		}
	}
	return err
}

// Contacts returns all available contacts.
func (db SQLLiteDatabase) Contacts() ([]Contact, error) {
	rows, err := db.db.Query("SELECT name, type, state, topic, public_key FROM user_contacts")
	if err != nil {
		return nil, err
	}

	var (
		rst = []Contact{}
		buf bytes.Buffer
	)
	for rows.Next() {
		// do not reuse same gob instance. same instance marshalls two same objects differently
		// if used repetitively.
		dec := gob.NewDecoder(&buf)
		contact := Contact{}
		pkey := []byte{}
		err = rows.Scan(&contact.Name, &contact.Type, &contact.State, &contact.Topic, &pkey)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			buf.Write(pkey)
			err = dec.Decode(&contact.PublicKey)
			if err != nil {
				return nil, err
			}
			buf.Reset()
		}
		rst = append(rst, contact)
	}
	return rst, nil
}

func (db SQLLiteDatabase) DeleteContact(c Contact) error {
	_, err := db.db.Exec("DELETE FROM user_contacts WHERE id = ?", fmt.Sprintf("%s:%d", c.Name, c.Type))
	if err != nil {
		return errors.Wrap(err, "error deleting contact from db")
	}
	return nil
}

func (db SQLLiteDatabase) PublicContactExist(c Contact) (exists bool, err error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if c.PublicKey != nil {
		err = enc.Encode(c.PublicKey)
		if err != nil {
			return false, err
		}
	} else {
		return false, errors.New("no public key")
	}
	err = db.db.QueryRow("SELECT EXISTS(SELECT id FROM user_contacts WHERE public_key = ?)", buf.Bytes()).Scan(&exists)
	return exists, err
}

func (db SQLLiteDatabase) LastMessageClock(c Contact) (int64, error) {
	var last sql.NullInt64
	err := db.db.QueryRow("SELECT max(clock) FROM user_messages WHERE contact_id = ?", contactID(c)).Scan(&last)
	if err != nil {
		return 0, err
	}
	return last.Int64, nil
}

func (db SQLLiteDatabase) SaveMessages(c Contact, messages []*protocol.Message) (last int64, err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return
	}
	stmt, err = tx.Prepare(`INSERT INTO user_messages(
id, contact_id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		} else {
			// don't shadow original error
			_ = tx.Rollback()
			return
		}
	}()

	var (
		buf       bytes.Buffer
		contactID = fmt.Sprintf("%s:%d", c.Name, c.Type)
		rst       sql.Result
	)
	for _, msg := range messages {
		enc := gob.NewEncoder(&buf)
		pkey := []byte{}
		if msg.SigPubKey != nil {
			err = enc.Encode(msg.SigPubKey)
			if err != nil {
				return
			}
			pkey = append(pkey, buf.Bytes()...)
			buf.Reset()
		}
		rst, err = stmt.Exec(
			msg.ID, contactID, msg.ContentT, msg.MessageT, msg.Text,
			msg.Clock, msg.Timestamp, msg.Content.ChatID, msg.Content.Text, pkey)
		if err != nil {
			if err.Error() == uniqueIDContstraint {
				err = ErrMsgAlreadyExist
			}
			return
		}
		last, err = rst.LastInsertId()
		if err != nil {
			return
		}
	}
	return
}

// Messages returns messages for a given contact, in a given period. Ordered by a timestamp.
func (db SQLLiteDatabase) Messages(c Contact, from, to time.Time) (result []*protocol.Message, err error) {
	contactID := fmt.Sprintf("%s:%d", c.Name, c.Type)
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key
FROM user_messages WHERE contact_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`,
		contactID, protocol.TimestampInMsFromTime(from), protocol.TimestampInMsFromTime(to))
	if err != nil {
		return nil, err
	}
	var (
		rst = []*protocol.Message{}
		buf bytes.Buffer
	)
	for rows.Next() {
		dec := gob.NewDecoder(&buf)
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			buf.Write(pkey)
			err = dec.Decode(&msg.SigPubKey)
			if err != nil {
				return nil, err
			}
			buf.Reset()
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

func (db SQLLiteDatabase) GetNewMessages(c Contact, rowid int64) ([]*protocol.Message, error) {
	contactID := fmt.Sprintf("%s:%d", c.Name, c.Type)
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key
FROM user_messages WHERE contact_id = ? AND rowid >= ? ORDER BY clock`,
		contactID, rowid)
	if err != nil {
		return nil, err
	}
	var (
		rst = []*protocol.Message{}
		buf bytes.Buffer
	)
	for rows.Next() {
		dec := gob.NewDecoder(&buf)
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			buf.Write(pkey)
			err = dec.Decode(&msg.SigPubKey)
			if err != nil {
				return nil, err
			}
			buf.Reset()
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

func contactID(c Contact) string {
	return fmt.Sprintf("%s:%d", c.Name, c.Type)
}
