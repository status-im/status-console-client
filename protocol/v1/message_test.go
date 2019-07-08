package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	testMessageBytes  = []byte(`["~#c4",["abc123","text/plain","~:public-group-user-message",154593077368201,1545930773682,["^ ","~:chat-id","testing-adamb","~:text","abc123"]]]`)
	testMessageStruct = Message{
		Text:      "abc123",
		ContentT:  "text/plain",
		MessageT:  "public-group-user-message",
		Clock:     154593077368201,
		Timestamp: 1545930773682,
		Content:   Content{"testing-adamb", "abc123"},
	}
)

func TestDecodeMessage(t *testing.T) {
	val, err := DecodeMessage(testMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t,
		&StatusMessage{
			Message: testMessageStruct,
		}, val)
}

func BenchmarkDecodeMessage(b *testing.B) {
	_, err := DecodeMessage(testMessageBytes)
	if err != nil {
		b.Fatalf("failed to decode message: %v", err)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _ = DecodeMessage(testMessageBytes)
	}
}

func TestEncodeMessage(t *testing.T) {
	data, err := EncodeMessage(testMessageStruct)
	require.NoError(t, err)
	// Decode it back to a struct because, for example, map encoding is non-deterministic
	// and it is not possible to compare bytes.
	val, err := DecodeMessage(data)
	require.NoError(t, err)
	require.EqualValues(t, &StatusMessage{Message: testMessageStruct}, val)
}

func TestTimestampInMs(t *testing.T) {
	ts := TimestampInMs(1555274502548) // random timestamp in milliseconds
	tt := ts.Time()
	require.Equal(t, tt.UnixNano(), 1555274502548*int64(time.Millisecond))
	require.Equal(t, ts, TimestampInMsFromTime(tt))
}
