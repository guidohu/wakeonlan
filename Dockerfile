# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o wakeonlan main.go

# Production stage
FROM alpine:latest

RUN addgroup -S wakeonlan && adduser -S wakeonlan -G wakeonlan
RUN apk add --no-cache iputils libcap
# Grant the binary permission to use raw sockets even for non-root users
RUN setcap cap_net_raw+ep /bin/ping || setcap cap_net_raw+ep /usr/bin/ping

WORKDIR /app

# Copy the binary and static files from the builder stage
COPY --from=builder /app/wakeonlan .
COPY static/ ./static/

RUN chown -R wakeonlan:wakeonlan /app

# Set up a directory for persistent storage
RUN mkdir -p /data && chown -R wakeonlan:wakeonlan /data
COPY --chown=wakeonlan:wakeonlan hosts.json.sample /data/hosts.json
ENV HOSTS_FILE=/data/hosts.json
ENV PORT=8080

USER wakeonlan

EXPOSE 8080

# Run the binary
CMD ["./wakeonlan"]
