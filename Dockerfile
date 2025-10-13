# Build stage
FROM golang:1.25.1-alpine AS builder
WORKDIR /app
RUN apt-get update && \
    apt-get install -y git && \
    rm -rf /var/lib/apt/lists/*
COPY go.mod .
COPY main.go .
RUN go mod tidy
RUN go build -o /telephone .

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /telephone .
EXPOSE 8080
CMD ["/app/telephone"]
