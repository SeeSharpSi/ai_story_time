# --- Builder Stage ---
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 builds a static binary
# -o /app/server builds the output to /app/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# --- Final Stage ---
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/server .

# Copy static files and templates
COPY static ./static
COPY templates ./templates
COPY data.db .

# Expose the port the app runs on
EXPOSE 9779

# Command to run the executable
CMD ["./server"]
