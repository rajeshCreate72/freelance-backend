# Build stage
FROM golang:1.27.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o freelance-server .


# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/freelance-server .

EXPOSE 8080

CMD ["./freelance-server"]
