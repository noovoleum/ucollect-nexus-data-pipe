# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o data-pipe ./cmd/data-pipe

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/data-pipe .

# Copy example config
COPY examples/config.json ./config.example.json

# Run the application
ENTRYPOINT ["./data-pipe"]
CMD ["-config", "config.json"]
