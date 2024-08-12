FROM golang:1.22.6 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sefud ./cmd/sefud

FROM debian:bullseye-slim

COPY --from=builder /app/sefud /usr/local/bin/sefud
EXPOSE 7000

ENTRYPOINT ["/usr/local/bin/sefud"]

