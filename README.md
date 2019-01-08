Status Console User Interface
=============================

**This is not an official Status client. It should be used exclusively for development purposes.**

The main motivation for writing this client is to have a second implementation of the messaging protocol in order to run protocol compatibility smoke tests. It will also allow us to iterate faster and test some approaches as eventually we want to move the whole messaging protocol details to [status-go](https://github.com/status-im/status-go).

At the same time, it's more powerful than relying on [Status Node](https://status.im/docs/run_status_node.html) JSON-RPC commands because it has direct access to the p2p server and the Whisper service.

# Start

```bash
# build a binary
$ go build -o ./bin/status-term-client .

# generate a private key
$ ./bin/status-term-client -create-key-pair
Your private key: <KEY>

# start
$ ./bin/status-term-client -keyhex=<KEY>

# or start and redirect logs
$ ./bin/status-term-client -keyhex=<KEY> 2>/tmp/status-term-client.log

# more options
$ ./bin/status-term-client -h
```

# Packages

The main package contains the console user interface.

* `github.com/status-im/status-term-client/protocol/v1` contains the current messaging protocol payload encoders and decoders as well as some utilities like creating a Whisper topic for a public chat.

# License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
