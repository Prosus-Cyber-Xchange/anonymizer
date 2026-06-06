FROM golang:1.25.0-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor/ vendor/

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="-s -w" -o /anonymizer-service ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /anonymizer-service /usr/local/bin/anonymizer-service

EXPOSE 8080

ENTRYPOINT ["anonymizer-service"]