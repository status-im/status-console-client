package datasync

import (
	"database/sql"

	"github.com/status-im/mvds"
)

type SyncState struct {
	db *sql.DB
}

func (s *SyncState) Get(group mvds.GroupID, id mvds.MessageID, sender mvds.PeerID) interface{} {
	panic("implement me")
}

func (s *SyncState) Set(group mvds.GroupID, id mvds.MessageID, sender mvds.PeerID, newState interface{}) {
	panic("implement me")
}

func (s *SyncState) Remove(group mvds.GroupID, id mvds.MessageID, sender mvds.PeerID) {
	s.db.Query("")
}

func (s *SyncState) Map(process func(mvds.GroupID, mvds.MessageID, mvds.PeerID, interface{}) interface{}) {
	panic("implement me")
}

