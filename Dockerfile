# --- Builder Stage ---
    FROM golang:1.23-alpine AS builder

    # Set necessary environment variables
    ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
    
    WORKDIR /app
    
    # Copy go.mod and go.sum first to leverage Docker layer caching
    COPY go.mod go.sum ./
    RUN go mod download
    
    # Copy the rest of the source code
    COPY . .
    
    # Build the static binary named 'server'
    RUN go build -o /bin/server ./cmd/server/main.go
    
    
    # --- Runtime Stage ---
    # Using alpine as a base. Consider distroless/static for even smaller/more secure images.
    FROM alpine:3.19
    
    # Install ca-certificates for HTTPS calls (if any) and timezone data
    # If your app connects to Postgres using TLS, ca-certificates is necessary.
    RUN apk update && apk add --no-cache ca-certificates tzdata && rm -rf /var/cache/apk/*
    
    # Copy the static binary from the builder stage
    COPY --from=builder /bin/server /usr/local/bin/server
    
    WORKDIR /app
    
    # Expose the port the application listens on (must match APP_PORT env var)
    EXPOSE 8080
    
    # Add non-root user for security
    RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    USER appuser
    
    # Command to run the application
    # The actual port and DB details will be injected via environment variables
    ENTRYPOINT ["/usr/local/bin/server"]