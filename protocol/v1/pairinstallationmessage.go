package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// PairInstallationMessage contains all message details.
type PairInstallationMessage struct {
	InstallationID string `json:"installationId"` // TODO: why is this duplicated?
	// The type of the device
	DeviceType string `json:"deviceType"`
	// Name the user set name
	Name string `json:"name"`
	// The FCMToken for mobile platforms
	FCMToken string `json:"fcmToken"`

	// not protocol defined fields
	ID []byte `json:"-"`
}

func (m *PairInstallationMessage) MarshalJSON() ([]byte, error) {
	type PairInstallationMessageAlias PairInstallationMessage
	item := struct {
		*PairInstallationMessageAlias
		ID string `json:"id"`
	}{
		PairInstallationMessageAlias: (*PairInstallationMessageAlias)(m),
		ID:                           fmt.Sprintf("%#x", m.ID),
	}

	return json.Marshal(item)
}

// CreatePairInstallationMessage creates a PairInstallationMessage
func CreatePairInstallationMessage(installationID string, name string, deviceType string, fcmToken string) PairInstallationMessage {
	return PairInstallationMessage{
		InstallationID: installationID,
		Name:           name,
		DeviceType:     deviceType,
		FCMToken:       fcmToken,
	}
}

// DecodeMessage decodes a raw payload to Message struct.
func DecodePairInstallationMessage(data []byte) (message PairInstallationMessage, err error) {
	buf := bytes.NewBuffer(data)
	decoder := NewMessageDecoder(buf)
	value, err := decoder.Decode()
	if err != nil {
		return
	}

	message, ok := value.(PairInstallationMessage)
	if !ok {
		return message, ErrInvalidDecodedValue
	}
	return
}

// EncodeMessage encodes a Message using Transit serialization.
func EncodePairInstallationMessage(value PairInstallationMessage) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
