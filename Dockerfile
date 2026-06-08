FROM golang:1.25.0-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="-s -w" -o /anonymizer ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /anonymizer /usr/local/bin/anonymizer

EXPOSE 8080

ENTRYPOINT ["anonymizer"]