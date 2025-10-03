# Build stage
FROM golang:1.19-alpine AS builder
WORKDIR /app
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
