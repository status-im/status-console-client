# status-nim

Status-(Go)-Nimbus interop.

Current setup: `nimbus -> expose bindings and shared library -> status-nim -> status-console-client`. Eventually this can expose a similar API as `status-go`, but for now interop with it is easier.

## Misc issues and how to solve them

Can't find `libnimbus_api.so`:

```
mv libnimbus_api.so /usr/local/lib/`
```

To run as a standalone process:

Change package name in `main.go` to `main`.

go-vendor issues:

```
> cp -r $GOPATH/src/github.com/status-im/status-nim/ $GOPATH/src/github.com/status-im/status-console-client/vendor/github.com/status-im/
```

Run from status-term-client:
```
# checkout nimbus-test branch/PR
make build

> ./bin/status-term-client -keyhex=0xe8b3b8a7cae540ace9bcaf6206e81387feb6415016aee75307976084f7751ed7 2>/tmp/status-term-client.log
```

Get predictable segfault (see `segfault.output`)

Replacing libnimbus lib for debugging info:

```
# Checkout nimbus branch status-c-api
# Run ./build_status_api.sh
# Copy over shared library to /usr/local/lib
# Ensure your go cache in clean, e.g. through `go clean -cache` so it is using new lib
```

Run through gdb like this to get stacktrace:

```
gdb --args ./bin/status-term-client -keyhex=0xe8b3b8a7cae540ace9bcaf6206e81387feb6415016aee75307976084f7751ed7 2>/tmp/status-term-client.log

# run
# Get seg fault
# bt
# => stacktrace
```

Stacktrace and current integration PRs:
https://gist.github.com/oskarth/771034417a52927fa9bbc6df415d5714
https://github.com/status-im/nimbus/pull/331#issuecomment-506308167
https://github.com/status-im/status-console-client/pull/79

With foreignthread GC hangs:
https://gist.github.com/oskarth/c3f4392e84c279c433474d31b3173737
https://github.com/status-im/nimbus/pull/331
