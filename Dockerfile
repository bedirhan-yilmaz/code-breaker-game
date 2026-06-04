# ---- Build stage ----
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy dependency manifests and download modules (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server /app/server

# Server listens on 8080
EXPOSE 8080

# Default: run the server.
# Override with: docker run ... /app/server client <host>
ENTRYPOINT ["/app/server"]
CMD ["server"]
