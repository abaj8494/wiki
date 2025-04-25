FROM golang:1.21-alpine AS builder

WORKDIR /app

# Force rebuild on every deployment
RUN echo "Rebuild timestamp: $(date)"

# Copy source code
COPY wiki.go backup.go ./
COPY edit.html view.html index.html ./

# Initialize a Go module and build the application
RUN go mod init wiki && go build -o wiki .

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /app

# Copy the binary and template files
COPY --from=builder /app/wiki /app/wiki
COPY --from=builder /app/edit.html /app/edit.html
COPY --from=builder /app/view.html /app/view.html
COPY --from=builder /app/index.html /app/index.html

# Create directories
RUN mkdir -p /app/files
RUN mkdir -p /app/persistent

# Expose the port
EXPOSE 21313

# Run the application
CMD ["/app/wiki"] 