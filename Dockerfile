# Build stage
FROM golang:1.25.2-3.22 AS builder
WORKDIR /app
RUN apk update && \
    apk add --no-cache git && 
COPY main.go .
RUN go mod tidy
RUN go build -o /telephone .

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /telephone .
EXPOSE 8080
CMD ["/app/telephone"]
