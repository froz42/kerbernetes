FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24 AS builder

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ARG VERSION=dev
ENV VERSION=${VERSION}

COPY . .

RUN go build -o kerbernetes-api -ldflags="-s -w -X main.Version=${VERSION}" ./cmd/api/main.go

FROM alpine:3.24.3

RUN adduser -D -g '' kerbernetes-api

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/kerbernetes-api /kerbernetes-api

USER kerbernetes-api

ENTRYPOINT ["/kerbernetes-api"]
