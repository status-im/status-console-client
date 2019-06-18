# Build status-go in a Go builder container
FROM golang:1.12.5-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

RUN mkdir -p /go/src/github.com/status-im/status-console-client
ADD . /go/src/github.com/status-im/status-console-client
WORKDIR /go/src/github.com/status-im/status-console-client
RUN make build

# Copy the binary to the second image
FROM alpine:latest

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-console-client"

COPY --from=builder /go/src/github.com/status-im/status-console-client/bin/status-term-client /usr/local/bin/

# 30304 is used for Discovery v5
EXPOSE 8080 8545 30303 30303/udp 30304/udp

ENTRYPOINT ["/usr/local/bin/status-term-client", "-no-ui"]
CMD ["--help"]