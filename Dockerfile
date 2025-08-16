FROM golang:1.24.6-alpine3.22 AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ARG VERSION=dev
ENV VERSION=${VERSION}

COPY . .

RUN go build -o kerbernetes-api -ldflags="-s -w -X main.Version=${VERSION}" ./cmd/api/main.go

FROM alpine:3.22

RUN adduser -D -g '' kerbernetes-api

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/kerbernetes-api /kerbernetes-api

USER kerbernetes-api

ENTRYPOINT ["/kerbernetes-api"]