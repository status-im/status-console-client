package datasync

import (
	"database/sql"

	"github.com/status-im/mvds"
)

type SyncState struct {
	db *sql.DB
}

func (s *SyncState) Get(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID) (mvds.State, error) {
	r, err := s.db.Query("SELECT * FROM state WHERE groupID = ? AND id = ? AND peer = ?", group[:], id[:], peer[:])
	if err != nil {
		return mvds.State{}, err
	}

	if !r.Next() {
		return mvds.State{}, err
	}

	var count uint64
	var epoch int64
	err = r.Scan(&count, &epoch)
	if err != nil {
		return mvds.State{}, err
	}

	return mvds.State{
		SendCount: count,
		SendEpoch: epoch,
	}, nil
}

func (s *SyncState) Set(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID, newState mvds.State) error {
	panic("implement me")
}

func (s *SyncState) Remove(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID) error {
	q, err := s.db.Prepare("DELETE FROM state WHERE groupID = ? AND id = ? AND peer = ?")
	if err != nil {
		// @todo
	}

	r, err := q.Exec(group[:], id[:], peer[:])
	if err != nil {
		return err
	}

	affected, err := r.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		// @todo
	}

	return nil
}

func (s *SyncState) Map(process func(mvds.GroupID, mvds.MessageID, mvds.PeerID, mvds.State) mvds.State) error {
	panic("implement me")
}
