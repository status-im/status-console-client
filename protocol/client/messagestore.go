package client

import "github.com/status-im/mvds"

type MessageStore struct {

}

func (ms *MessageStore) HasMessage(id mvds.MessageID) bool {
	panic("implement me")
}

func (ms *MessageStore) GetMessage(id mvds.MessageID) (mvds.Message, error) {
	panic("implement me")
}

func (ms *MessageStore) SaveMessage(message mvds.Message) error {

	// @todo we probably want to decode the body here so we have it for later usage
	// this should allow us to replace the current message database

	panic("implement me")
}
