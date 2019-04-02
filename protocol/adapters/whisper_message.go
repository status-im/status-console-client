package adapters

import (
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-console-client/protocol/v1"
	whisper "github.com/status-im/whisper/whisperv6"
)

func createWhisperNewMessage(data []byte, sigKey string) whisper.NewMessage {
	return whisper.NewMessage{
		TTL:       WhisperTTL,
		Payload:   data,
		PowTarget: WhisperPoW,
		PowTime:   WhisperPoWTime,
		Sig:       sigKey,
	}
}

func setEncryptionKeyForNewMessage(message *whisper.NewMessage, keys keysManager, options protocol.SendOptions) (err error) {
	if options.Recipient != nil {
		message.PublicKey = crypto.FromECDSAPub(options.Recipient)
		return
	}

	if options.ChatName != "" {
		message.SymKeyID, err = keys.AddOrGetSymKeyFromPassword(options.ChatName)
		if err != nil {
			return
		}
	}

	return errors.New("failed to set an encryption key")
}

func createRichWhisperNewMessage(keys keysManager, data []byte, options protocol.SendOptions) (whisper.NewMessage, error) {
	var message whisper.NewMessage

	sigKey, err := keys.AddOrGetKeyPair(options.Identity)
	if err != nil {
		return message, err
	}

	message = createWhisperNewMessage(data, sigKey)

	message.Topic, err = topicForSendOptions(options)
	if err != nil {
		return message, err
	}

	err = setEncryptionKeyForNewMessage(&message, keys, options)
	return message, err
}
