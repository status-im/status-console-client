protocol
========

It contains the Status protocol implementation in Go.

It is divided into three packages:
* `adapters` contains a code that allows sending and receiving Status protocol messages through the Whisper network,
* `client` contains high-level abstraction over the protocol which includes handling contacts and messages,
* `v1` is the current protocol low-level implementation which includes things like encoding and decoding a message payload as well as `Chat` interface that should be implemented by adapters.
