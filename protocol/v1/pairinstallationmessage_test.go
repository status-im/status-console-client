package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testPairInstallationMessageBytes  = []byte(`["~#p2",["installation-id","desktop","name","token"]]`)
	testPairInstallationMessageStruct = PairInstallationMessage{
		Name:           "name",
		DeviceType:     "desktop",
		FCMToken:       "token",
		InstallationID: "installation-id",
	}
)

func TestDecodePairInstallationMessageMessage(t *testing.T) {
	val, err := DecodeMessage(testPairInstallationMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t, testPairInstallationMessageStruct, val)
}

func TestEncodePairInstallationMessage(t *testing.T) {
	data, err := EncodePairInstallationMessage(testPairInstallationMessageStruct)
	require.NoError(t, err)
	// Decode it back to a struct because, for example, map encoding is non-deterministic
	// and it is not possible to compare bytes.
	val, err := DecodeMessage(data)
	require.NoError(t, err)
	require.EqualValues(t, testPairInstallationMessageStruct, val)
}
