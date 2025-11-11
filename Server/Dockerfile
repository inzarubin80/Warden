# -------- Builder --------
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git build-base ca-certificates

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

# -------- Runtime --------
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /out/server /usr/local/bin/server

# Non-root user
USER 65532:65532

EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/server"]


