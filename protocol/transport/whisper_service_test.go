package transport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectAndAddNoMailservers(t *testing.T) {
	svc := &WhisperServiceTransport{}
	rst, err := svc.selectAndAddMailServer()
	require.Empty(t, rst)
	require.EqualError(t, ErrNoMailservers, err.Error())
}
