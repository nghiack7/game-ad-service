FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk --no-cache add ca-certificates git

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./main.go

# Use a minimal alpine image for the final container
FROM alpine:3.18

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the built executables from the builder stage

COPY --from=builder /app/server ./server
COPY --from=builder /app/config.yaml ./config.yaml
COPY --from=builder /app/server ./worker
COPY --from=builder /app/config.yaml ./config.yaml

# Set user to non-root
USER appuser

# Expose port
EXPOSE 8080

# Command to run the application (set at runtime)
CMD ["./server", "server"]
