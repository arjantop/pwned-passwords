FROM golang:1.10-alpine

RUN apk add --update openssl git && \
    wget -O /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 && \
    chmod +x /usr/local/bin/dep
WORKDIR /go/src/github.com/arjantop/pwned-passwords/
COPY . .
RUN dep ensure
RUN go build -o main server/server.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/arjantop/pwned-passwords/main server

ENTRYPOINT ["./server"]
