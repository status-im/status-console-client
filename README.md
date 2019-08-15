Status Console User Interface
=============================

**This is not an official Status client. It should be used exclusively for development purposes.**

The main motivation for writing this client is to have a second implementation of the messaging protocol in order to run protocol compatibility smoke tests. It will also allow us to iterate faster and test some approaches as eventually we want to move the whole messaging protocol details to [status-go](https://github.com/status-im/status-go).

At the same time, it's more powerful than relying on [Status Node](https://status.im/docs/run_status_node.html) JSON-RPC commands because it has direct access to the p2p server and the Whisper service.

# Start

```bash
# build a binary
$ make build

# generate a private key
$ ./bin/status-term-client -create-key-pair
Your private key: <KEY>

# start
$ ./bin/status-term-client -keyhex=<KEY> -installation-id=any-string -data-dir=your-data-dir

# or start and redirect logs
$ ./bin/status-term-client -keyhex=<KEY> 2>/tmp/status-term-client.log

# more options
$ ./bin/status-term-client -h
```

# Commands

Commands starts with `/` and must be typed in the INPUT view in the UI.

Currently the following commands are supported.

## Adding a public chat

`/chat add <topic>`

## Adding a contact

`/chat add <public-key> <name>`

# Packages

The main package contains the console user interface.

* `github.com/status-im/status-console-client/protocol/v1` contains the current messaging protocol payload encoders and decoders as well as some utilities like creating a Whisper topic for a public chat.

# (Very) Experimental Nimbus support

`status-console-client` supports very experimental Nimbus support for Whisper.

## How it works

1. Nimbus exposes a basic and very rough C API for Whisper polling, posting,
  subscribing, and adding peers: https://github.com/status-im/nimbus/pull/331

2. This C API is consumed as a standard shared library, `libnimbus_api.so`.

3. [status-nim](https://github.com/status-im/status-nim) wraps this library to
expose a Go API. Currently, this "API" is more like a hacky spike. The goal is
for this to library to hide the integration details with Nim and provide a clean
Go interface for consumers.
 
## Building and running

The changes are isolated and won't impact `status-console-client` unless the
appropriate build instruction, flag and patch is provided.

### libnimbus_api.so

If you have issues with `libnimbus_api.so` (likely) you might want to copy it
into `/usr/local/lib` manually.

```
make build-nimbus
./bin/status-term-client -keyhex=0x9af3cdb76d76da2b36d2dcc082cb54ea672639331ef03b91a62ad6ef804b4896 -nimbus
```

Expected output:

```
[nim-status] posting ["~#c4",["Message:1000","text/plain","~:public-group-user-message",156393648280100,1563936482801,["^ ","~:chat-id","status-test-c","~:text","Message:1000"]]]
```

And a message being posted in `#status-test-c`.


# License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
