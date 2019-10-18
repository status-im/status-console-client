// +build nimbus,!geth

package main

import (
	"crypto/ecdsa"

	status "github.com/status-im/status-protocol-go"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
)

const noGethError = "executable needs to be built without -tags nimbus or with -tags geth"

func newGethWhisperWrapper(pk *ecdsa.PrivateKey) (whispertypes.Whisper, error) {
	panic(noGethError)
}

func createMessengerWithURI(uri string) (*status.Messenger, error) {
	panic(noGethError)
}
