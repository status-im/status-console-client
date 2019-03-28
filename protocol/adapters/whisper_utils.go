package adapters

import (
	"math/rand"

	whisper "github.com/status-im/whisper/whisperv6"
)

// shhextRequestMessagesParam is used to remove dependency on shhext module.
type shhextRequestMessagesParam struct {
	MailServerPeer string              `json:"mailServerPeer"`
	From           int64               `json:"from"`
	To             int64               `json:"to"`
	Limit          int                 `json:"limit"`
	SymKeyID       string              `json:"symKeyID"`
	Topics         []whisper.TopicType `json:"topics"`
}

func createWhisperNewMessage(data []byte, sigKey string) whisper.NewMessage {
	return whisper.NewMessage{
		TTL:       60,
		Payload:   data,
		PowTarget: 2.0,
		PowTime:   5,
		Sig:       sigKey,
	}
}

func randomItem(items []string) string {
	l := len(items)
	return items[rand.Intn(l)]
}
