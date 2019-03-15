package protocol

import (
	whisper "github.com/status-im/whisper/whisperv6"
	"golang.org/x/crypto/sha3"
)

// PublicChatTopic returns a Whisper topic for a public channel name.
func PublicChatTopic(name string) (whisper.TopicType, error) {
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write([]byte(name)); err != nil {
		return whisper.TopicType{}, err
	}

	return whisper.BytesToTopic(hash.Sum(nil)), nil
}

// PrivateChatTopic returns a Whisper topic for a private chat.
func PrivateChatTopic() (whisper.TopicType, error) {
	return PublicChatTopic("contact-discovery")
}
