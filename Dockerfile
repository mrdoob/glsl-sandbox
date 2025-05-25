FROM golang:1.24 AS builder

ENV GOPATH /go
RUN apt-get update && \
    apt-get install -y libsqlite3-0 libsqlite3-dev

WORKDIR /build
COPY . .

RUN go build -tags cgosqlite -v ./server/cmd/glslsandbox

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

EXPOSE 8888
EXPOSE 8883
COPY --from=builder /build/ /glslsandbox/
ENTRYPOINT [ "/glslsandbox/entrypoint.sh" ]
