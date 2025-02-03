FROM golang:1.23.5 AS builder

WORKDIR /app
COPY . .

RUN go build -o grafana-users-provision .

# Use a minimal base image to run the app
FROM alpine:3.21.2

# Copy binary from the builder stage
COPY --from=builder /app/grafana-users-provision /usr/local/bin/grafana-users-provision

ENTRYPOINT ["/usr/local/bin/grafana-users-provision"]
