package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/status-im/status-console-client/protocol/client/migrations"
	"github.com/status-im/status-console-client/protocol/client/sqlite"
	"github.com/status-im/status-console-client/protocol/v1"
)

func marshalEcdsaPub(pub *ecdsa.PublicKey) (rst []byte, err error) {
	switch pub.Curve.(type) {
	case *secp256k1.BitCurve:
		rst = make([]byte, 34)
		rst[0] = 1
		copy(rst[1:], secp256k1.CompressPubkey(pub.X, pub.Y))
		return rst[:], nil
	default:
		return nil, errors.New("unknown curve")
	}
}

func unmarshalEcdsaPub(buf []byte) (*ecdsa.PublicKey, error) {
	pub := &ecdsa.PublicKey{}
	if len(buf) < 1 {
		return nil, errors.New("too small")
	}
	switch buf[0] {
	case 1:
		pub.Curve = secp256k1.S256()
		pub.X, pub.Y = secp256k1.DecompressPubkey(buf[1:])
		ok := pub.IsOnCurve(pub.X, pub.Y)
		if !ok {
			return nil, errors.New("not on curve")
		}
		return pub, nil
	default:
		return nil, errors.New("unknown curve")
	}
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
	Messages(c Chat, from, to time.Time) ([]*protocol.Message, error)
	NewMessages(c Chat, rowid int64) ([]*protocol.Message, error)
	UnreadMessages(c Chat) ([]*protocol.Message, error)
	SaveMessages(c Chat, messages []*protocol.Message) (int64, error)
	LastMessageClock(Chat) (int64, error)
	Chats() ([]Chat, error)
	SaveChats(chats []Chat) error
	DeleteChat(Chat) error
	ChatExist(Chat) (bool, error)
	GetPublicChat(name string) (*Chat, error)
	GetOneToOneChat(*ecdsa.PublicKey) (*Chat, error)
}

// Migrate applies migrations.
func Migrate(db *sql.DB) error {
	resources := bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	log.Printf("[Migrate] applying migrations %s", migrations.AssetNames())

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
	db, err := InitializeDB(tmpfile.Name(), hex.EncodeToString(pass))
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

// SaveChats inserts or replaces provided chats.
// TODO should it delete all previous chats?
func (db SQLLiteDatabase) SaveChats(chats []Chat) (err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
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
	stmt, err = tx.Prepare(`INSERT INTO chats(
	  id,
	  name,
	  color,
	  type,
	  active,
	  updated_at,
	  deleted_at_clock_value,
	  public_key,
	  unviewed_message_count,
	  last_clock_value,
	  last_message_content_type,
	  last_message_content
	)
	  VALUES
	  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for i := range chats {
		pkey := []byte{}
		if chats[i].PublicKey != nil {
			pkey, err = marshalEcdsaPub(chats[i].PublicKey)
			if err != nil {
				return err
			}
		}
		id := chatID(chats[i])
		_, err = stmt.Exec(id,
			chats[i].Name,
			chats[i].Color,
			chats[i].Type,
			chats[i].Active,
			chats[i].UpdatedAt,
			chats[i].DeletedAtClockValue,
			pkey,
			chats[i].UnviewedMessageCount,
			chats[i].LastClockValue,
			chats[i].LastMessageContentType,
			chats[i].LastMessageContent,
		)
		if err != nil {
			return err
		}
	}
	return err
}

// Chats returns all available chats.
func (db SQLLiteDatabase) Chats() ([]Chat, error) {
	rows, err := db.db.Query("SELECT name, type, public_key FROM chats")
	if err != nil {
		return nil, err
	}

	var (
		rst = []Chat{}
	)
	for rows.Next() {
		// do not reuse same gob instance. same instance marshalls two same objects differently
		// if used repetitively.
		chat := Chat{}
		pkey := []byte{}
		err = rows.Scan(&chat.Name, &chat.Type, &pkey)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			chat.PublicKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, chat)
	}
	return rst, nil
}

func (db SQLLiteDatabase) DeleteChat(c Chat) error {
	_, err := db.db.Exec("DELETE FROM chats WHERE id = ?", fmt.Sprintf("%s:%d", c.Name, c.Type))
	if err != nil {
		return errors.Wrap(err, "error deleting chat from db")
	}
	return nil
}

func (db SQLLiteDatabase) ChatExist(c Chat) (exists bool, err error) {
	err = db.db.QueryRow("SELECT EXISTS(SELECT id FROM chats WHERE id = ?)", chatID(c)).Scan(&exists)
	return
}

func (db SQLLiteDatabase) GetOneToOneChat(publicKey *ecdsa.PublicKey) (*Chat, error) {
	if publicKey == nil {
		return nil, errors.New("No public key provided")
	}

	pkey, err := marshalEcdsaPub(publicKey)
	if err != nil {
		return nil, err
	}

	c := &Chat{}
	err = db.db.QueryRow("SELECT name FROM chats WHERE public_key = ?", pkey).Scan(&c.Name)
	c.Type = OneToOneChat
	c.PublicKey = publicKey
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return c, nil
}

func (db SQLLiteDatabase) GetPublicChat(name string) (*Chat, error) {
	c := &Chat{}
	err := db.db.QueryRow("SELECT name FROM chats WHERE id = ?", formatID(name, PublicChat)).Scan(&c.Name)
	c.Type = PublicChat
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return c, nil
}

func (db SQLLiteDatabase) LastMessageClock(c Chat) (int64, error) {
	var last sql.NullInt64
	err := db.db.QueryRow("SELECT max(clock) FROM user_messages WHERE chat_id = ?", chatID(c)).Scan(&last)
	if err != nil {
		return 0, err
	}
	return last.Int64, nil
}

func (db SQLLiteDatabase) SaveMessages(c Chat, messages []*protocol.Message) (last int64, err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return
	}
	stmt, err = tx.Prepare(`INSERT INTO user_messages(
id, chat_id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
		chatID = fmt.Sprintf("%s:%d", c.Name, c.Type)
		rst    sql.Result
	)
	for _, msg := range messages {
		pkey := []byte{}
		if msg.SigPubKey != nil {
			pkey, err = marshalEcdsaPub(msg.SigPubKey)
		}
		rst, err = stmt.Exec(
			msg.ID, chatID, msg.ContentT, msg.MessageT, msg.Text,
			msg.Clock, msg.Timestamp, msg.Content.ChatID, msg.Content.Text,
			pkey, msg.Flags)
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

// Messages returns messages for a given chat, in a given period. Ordered by a timestamp.
func (db SQLLiteDatabase) Messages(c Chat, from, to time.Time) (result []*protocol.Message, err error) {
	chatID := fmt.Sprintf("%s:%d", c.Name, c.Type)
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags
FROM user_messages WHERE chat_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`,
		chatID, protocol.TimestampInMsFromTime(from), protocol.TimestampInMsFromTime(to))
	if err != nil {
		return nil, err
	}
	var (
		rst = []*protocol.Message{}
	)
	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

func (db SQLLiteDatabase) NewMessages(c Chat, rowid int64) ([]*protocol.Message, error) {
	chatID := chatID(c)
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags
FROM user_messages WHERE chat_id = ? AND rowid >= ? ORDER BY clock`,
		chatID, rowid)
	if err != nil {
		return nil, err
	}
	var (
		rst = []*protocol.Message{}
	)
	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

// TODO(adam): refactor all message getters in order not to
// repeat the select fields over and over.
func (db SQLLiteDatabase) UnreadMessages(c Chat) ([]*protocol.Message, error) {
	chatID := chatID(c)
	rows, err := db.db.Query(`
		SELECT
			id,
			content_type,
			message_type,
			text,
			clock,
			timestamp,
			content_chat_id,
			content_text,
			public_key,
			flags
		FROM
			user_messages
		WHERE
			chat_id = ? AND
			flags & ? == 0
		ORDER BY clock`,
		chatID, protocol.MessageRead,
	)
	if err != nil {
		return nil, err
	}

	var result []*protocol.Message

	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, &msg)
	}

	return result, nil
}

func chatID(c Chat) string {
	return formatID(c.Name, c.Type)
}

func formatID(name string, t ChatType) string {
	return fmt.Sprintf("%s:%d", name, t)
}
