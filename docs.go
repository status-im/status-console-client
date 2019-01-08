package main

import "crypto/ecdsa"

// Identity is an identification of the current user.
// It is required to send messages.
// In order to not repeat the complex type and allow easier refactoring,
// it's defined as a type alias.
type Identity = *ecdsa.PrivateKey
