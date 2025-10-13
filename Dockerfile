# Build stage
FROM golang:1.25.2-3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o telephone ./main.go


# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/telephone /app/telephone
EXPOSE 8080
CMD ["/app/telephone"]
