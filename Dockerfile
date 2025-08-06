FROM golang:1.23.0 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sefud ./cmd/sefud

FROM debian:bullseye-slim

COPY --from=builder /app/sefud /usr/local/bin/sefud

ENTRYPOINT ["/usr/local/bin/sefud"]

