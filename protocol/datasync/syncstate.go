package datasync

import (
	"database/sql"
	"log"

	"github.com/status-im/mvds/state"
)

type SyncState struct {
	db *sql.DB
}

func (s *SyncState) Get(group state.GroupID, id state.MessageID, peer state.PeerID) (state.State, error) {
	r, err := s.db.Query(
		"SELECT send_count, send_epoch FROM state WHERE group = ? AND id = ? AND peer = ?",
		group[:],
		id[:],
		peer[:],
	)

	if err != nil {
		return state.State{}, err
	}

	if !r.Next() {
		return state.State{}, nil
	}

	var count uint64
	var epoch int64
	err = r.Scan(&count, &epoch)
	if err != nil {
		return state.State{}, err
	}

	return state.State{
		SendCount: count,
		SendEpoch: epoch,
	}, nil
}

func (s *SyncState) Set(group state.GroupID, id state.MessageID, peer state.PeerID, newState state.State) error {
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

func (s *SyncState) Remove(group state.GroupID, id state.MessageID, peer state.PeerID) error {
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

func (s *SyncState) Map(epoch int64, process func(state.GroupID, state.MessageID, state.PeerID, state.State) state.State) error {
	r, err := s.db.Query("SELECT group, id, peer, send_count, send_epoch FROM state")
	if err != nil {
		return err
	}

	var (
		group []byte
		id    []byte
		peer  []byte
	)

	for r.Next() {
		state := state.State{}
		err = r.Scan(&group, &id, &peer, &state.SendCount, &state.SendEpoch)
		if err != nil {
			// @todo
			continue
		}

		g := groupID(group)
		m := messageID(id)
		p := peerID(peer)

		newState := process(g, m, p, state)
		if newState == state {
			continue
		}

		err = s.Set(g, m, p, newState)
		if err != nil {
			log.Printf("error while setting new state %s", err.Error())
		}
	}

	return nil
}

func groupID(bytes []byte) state.GroupID {
	id := state.GroupID{}
	copy(id[:], bytes)

	return id
}

func messageID(bytes []byte) state.MessageID {
	id := state.MessageID{}
	copy(id[:], bytes)

	return id
}

func peerID(bytes []byte) state.PeerID {
	id := state.PeerID{}
	copy(id[:], bytes)

	return id
}
