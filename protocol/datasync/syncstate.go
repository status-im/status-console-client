package datasync

import (
	"database/sql"

	"github.com/status-im/mvds"
)

type SyncState struct {
	db *sql.DB
}

func (s *SyncState) Get(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID) (mvds.State, error) {
	r, err := s.db.Query(
		"SELECT send_count, send_epoch FROM state WHERE group = ? AND id = ? AND peer = ?",
		group[:],
		id[:],
		peer[:],
	)

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
	q, err := s.db.Prepare("INSERT INTO state(group, id, peer, send_count, send_epoch) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = q.Exec(group[:], id[:], peer[:], newState.SendCount, newState.SendEpoch)
	if err != nil {
		return err
	}

	return nil
}

func (s *SyncState) Remove(group mvds.GroupID, id mvds.MessageID, peer mvds.PeerID) error {
	q, err := s.db.Prepare("DELETE FROM state WHERE group = ? AND id = ? AND peer = ?")
	if err != nil {
		return err
	}

	_, err = q.Exec(group[:], id[:], peer[:])
	if err != nil {
		return err
	}

	return nil
}

func (s *SyncState) Map(process func(mvds.GroupID, mvds.MessageID, mvds.PeerID, mvds.State) mvds.State) error {
	r, err := s.db.Query("SELECT group, id, peer, send_count, send_epoch FROM state")
	if err != nil {
		return err
	}

	var group []byte
	var id []byte
	var peer []byte
	for r.Next() {

		state := mvds.State{}

		err = r.Scan(&group, &id, &peer, &state.SendCount, &state.SendEpoch)
		if err != nil {
			// @todo
			continue
		}

		newState := process(groupID(group), messageID(id), peerID(peer), state)
		if newState == state {
			continue
		}

		// @todo set state
	}

	return nil
}

func groupID(bytes []byte) mvds.GroupID {
	id := mvds.GroupID{}
	copy(id[:], bytes)

	return id
}

func messageID(bytes []byte) mvds.MessageID {
	id := mvds.MessageID{}
	copy(id[:], bytes)

	return id
}

func peerID(bytes []byte) mvds.PeerID {
	id := mvds.PeerID{}
	copy(id[:], bytes)

	return id
}