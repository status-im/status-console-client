package main

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
)

type sqlitePersistence struct {
	db *sql.DB
}

func newSQLitePersistence(db *sql.DB) *sqlitePersistence {
	return &sqlitePersistence{db: db}
}

func (s *sqlitePersistence) Chats() ([]Chat, error) {
	rows, err := s.db.Query(
		`SELECT
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
		FROM chats`,
	)
	if err != nil {
		return nil, err
	}

	var rst []Chat

	for rows.Next() {
		// do not reuse same gob instance. same instance marshalls two same objects differently
		// if used repetitively.
		var (
			chat      Chat
			pkey      []byte
			updatedAt time.Time
		)

		err = rows.Scan(
			&chat.id,
			&chat.Name,
			&chat.Color,
			&chat.Type,
			&chat.Active,
			&updatedAt,
			&chat.DeletedAtClockValue,
			&pkey,
			&chat.UnviewedMessageCount,
			&chat.LastClockValue,
			&chat.LastMessageContentType,
			&chat.LastMessageContent,
		)
		if err != nil {
			return nil, err
		}
		chat.UpdatedAt = updatedAt.Unix()
		if len(pkey) != 0 {
			chat.publicKey, err = unmarshalECDSAPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, chat)
	}
	return rst, nil
}

func (s *sqlitePersistence) ChatExist(c Chat) (exists bool, err error) {
	err = s.db.QueryRow("SELECT EXISTS(SELECT id FROM chats WHERE id = ?)", c.ID()).Scan(&exists)
	return
}

func (s *sqlitePersistence) AddChats(chats ...Chat) (err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = s.db.BeginTx(context.Background(), &sql.TxOptions{})
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
		if chats[i].publicKey != nil {
			pkey, err = marshalECDSAPub(chats[i].publicKey)
			if err != nil {
				return err
			}
		}
		_, err = stmt.Exec(
			chats[i].ID(),
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

func (s *sqlitePersistence) DeleteChat(c Chat) error {
	_, err := s.db.Exec("DELETE FROM chats WHERE id = ?", fmt.Sprintf("%s:%d", c.Name, c.Type))
	if err != nil {
		return errors.Wrap(err, "error deleting chat from db")
	}
	return nil
}

func marshalECDSAPub(pub *ecdsa.PublicKey) (rst []byte, err error) {
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

func unmarshalECDSAPub(buf []byte) (*ecdsa.PublicKey, error) {
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
