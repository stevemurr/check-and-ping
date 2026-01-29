FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (needed for SQLite)
RUN apk add --no-cache gcc musl-dev

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with CGO enabled for SQLite support
RUN CGO_ENABLED=1 go build -o /checkandping ./cmd/checkandping

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /checkandping /checkandping

ENTRYPOINT ["/checkandping"]
CMD ["--config", "/config.yaml"]
