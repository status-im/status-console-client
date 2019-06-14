protocol
========

It contains the Status protocol implementation in Go.

It is divided into three packages:
* `adapter` contains a code that ties together the Status protocol and transport protocol like Whisper. It allows to inject another layers between decoding user messages and the transport as well.
* `client` contains high-level abstraction over the protocol which includes handling contacts and messages.
* `v1` is the current protocol low-level implementation of the message format as well as decoding and encoding functions.
