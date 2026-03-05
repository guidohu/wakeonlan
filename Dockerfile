# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY main.go ./

RUN go build -o wakeonlan main.go

# Production stage
FROM alpine:latest

WORKDIR /app

# Copy the binary and static files from the builder stage
COPY --from=builder /app/wakeonlan .
COPY static/ ./static/

# Set up a directory for persistent storage
RUN mkdir -p /data
COPY hosts.json.sample /data/hosts.json
ENV HOSTS_FILE=/data/hosts.json
ENV PORT=8080

EXPOSE 8080

# Run the binary
CMD ["./wakeonlan"]
