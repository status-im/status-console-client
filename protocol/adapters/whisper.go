package adapters

import "crypto/ecdsa"

type keysManager interface {
	AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error)
	AddOrGetSymKeyFromPassword(password string) (string, error)
	GetRawSymKey(string) ([]byte, error)
}
