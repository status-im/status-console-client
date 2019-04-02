package adapters

import (
	"errors"

	"github.com/status-im/status-console-client/protocol/v1"
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
	return PublicChatTopic(TopicDiscovery)
}

func topicForSendOptions(options protocol.SendOptions) (whisper.TopicType, error) {
	if options.Recipient != nil {
		return PrivateChatTopic()
	}

	if options.ChatName != "" {
		return PublicChatTopic(options.ChatName)
	}

	return whisper.TopicType{}, errors.New("invalid options")
}

func topicForSubscribeOptions(options protocol.SubscribeOptions) (whisper.TopicType, error) {
	if options.Identity != nil {
		return PrivateChatTopic()
	}

	if options.ChatName != "" {
		return PublicChatTopic(options.ChatName)
	}

	return whisper.TopicType{}, errors.New("invalid options")
}

func topicForRequestOptions(options protocol.RequestOptions) (whisper.TopicType, error) {
	if options.Recipient != nil {
		return PrivateChatTopic()
	}

	if options.ChatName != "" {
		return PublicChatTopic(options.ChatName)
	}

	return whisper.TopicType{}, errors.New("invalid options")
}
