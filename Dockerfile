FROM golang:1.23.0 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o sefud ./cmd/sefud

FROM golang:alpine

COPY --from=builder /app/sefud /usr/local/bin/sefud

ENTRYPOINT ["/usr/local/bin/sefud"]

