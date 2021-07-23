FROM golang:1.16 AS builder

ENV GOPATH /go
RUN apt-get update && \
    apt-get install -y libsqlite3-0 libsqlite3-dev

WORKDIR /build
COPY . .

RUN go build -v ./server/cmd/glslsandbox

FROM debian:buster-slim

EXPOSE 8888
EXPOSE 8883
COPY --from=builder /build/ /glslsandbox/
ENTRYPOINT [ "/glslsandbox/entrypoint.sh" ]
