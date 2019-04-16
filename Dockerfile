FROM golang:1.12-alpine

RUN apk add --no-cache git
WORKDIR /app-src/
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o /app-out/server server/server.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /app/
COPY --from=0 /app-out/server server

ENTRYPOINT ["./server"]
