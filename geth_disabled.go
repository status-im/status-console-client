// +build nimbus,!geth

package main

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol"
)

const noGethError = "executable needs to be built without -tags nimbus or with -tags geth"

func newGethWhisperWrapper(pk *ecdsa.PrivateKey) (types.Whisper, error) {
	panic(noGethError)
}

func createMessengerWithURI(uri string) (*protocol.Messenger, error) {
	panic(noGethError)
}
