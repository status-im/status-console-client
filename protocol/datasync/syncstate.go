package datasync

import (
	"database/sql"

	"github.com/status-im/mvds"
)

type SyncState struct {
	db *sql.DB
}

// @todo these functions should probably all return errors
func (s *SyncState) Get(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID) mvds.State {
	r, err := s.db.Query("SELECT * FROM state WHERE groupID = ? AND id = ? AND peer = ?", group[:], id[:], peer[:])
	if err != nil {
		// @todo
		return mvds.State{}
	}

	if !r.Next() {
		return mvds.State{}
	}

	var count   uint64
	var epoch   int64
	err = r.Scan(&count, &epoch)
	if err != nil {
		// @todo
		return mvds.State{}
	}

	return mvds.State{
		SendCount: count,
		SendEpoch: epoch,
	}
}

func (s *SyncState) Set(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID, newState mvds.State) {
	panic("implement me")
}

func (s *SyncState) Remove(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID) {
	q, err := s.db.Prepare("DELETE FROM state WHERE groupID = ? AND id = ? AND peer = ?")
	if err != nil {
		// @todo
		return
	}

	r, err := q.Exec(group[:], id[:], peer[:])
	if err != nil {
		// @todo
		return
	}

	// @todo check r.RowsAffected
	affected, err := r.RowsAffected()
	if err != nil {
		// @todo
		return
	}

	if affected != 1 {
		// @todo
	}
}

func (s *SyncState) Map(process func(mvds.GroupID, mvds.MessageID, mvds.PeerID, mvds.State) mvds.State) {
	panic("implement me")
}

