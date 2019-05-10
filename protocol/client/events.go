package client

import "github.com/status-im/status-console-client/protocol/v1"

type DatabaseWithEvents struct {
	Database
	feed chan<- interface{}
}

func NewDatabaseWithEvents(db Database, feed chan<- interface{}) DatabaseWithEvents {
	return DatabaseWithEvents{Database: db, feed: feed}
}

func (db DatabaseWithEvents) SaveMessages(c Contact, msgs []*protocol.Message) (int64, error) {
	rowid, err := db.Database.SaveMessages(c, msgs)
	if err != nil {
		return rowid, err
	}
	for _, m := range msgs {
		db.feed <- messageEvent{
			baseEvent: baseEvent{
				Contact: c,
				Type:    EventTypeMessage,
			},
			Message: m,
		}
	}
	return rowid, err
}
