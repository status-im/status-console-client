package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testMessageBytes  = []byte(`["~#c4",["abc123","text/plain","~:public-group-user-message",154593077368201,1545930773682,["^ ","~:chat-id","testing-adamb","~:text","abc123"]]]`)
	testMessageStruct = StatusMessage{
		Text:      "abc123",
		ContentT:  "text/plain",
		MessageT:  "public-group-user-message",
		Clock:     154593077368201,
		Timestamp: 1545930773682,
		Content:   StatusMessageContent{"testing-adamb", "abc123"},
	}
)

func TestDecodeMessage(t *testing.T) {
	val, err := DecodeMessage(testMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t, testMessageStruct, val)
}

func TestEncodeMessage(t *testing.T) {
	data, err := EncodeMessage(testMessageStruct)
	require.NoError(t, err)
	// Decode it back to a struct because, for example, map encoding is non-deterministic
	// and it is not possible to compare bytes.
	val, err := DecodeMessage(data)
	require.NoError(t, err)
	require.EqualValues(t, testMessageStruct, val)
}
