package transport

import "github.com/status-im/status-go/eth-node/types"

func DefaultMessage() types.NewMessage {
	msg := types.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.001
	msg.PowTime = 1

	return msg
}
