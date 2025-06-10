# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
# Output the binary to /go-churn-agent (within the builder stage)
RUN go build -o /go-churn-agent .

# Runtime Stage
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /go-churn-agent /app/go-churn-agent

# Ensure the binary is executable (alpine's default umask might not always set this)
RUN chmod +x /app/go-churn-agent

# Document the port the application listens on
EXPOSE 8080

# Document required environment variables (actual values should be provided at runtime)
ENV SUPABASE_URL=""
ENV SUPABASE_KEY=""

# Set the entrypoint to run the application
ENTRYPOINT ["/app/go-churn-agent"]
