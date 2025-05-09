FROM golang:1.23 AS builder
WORKDIR /app/media-service
COPY media-service/go.mod media-service/go.sum ./
RUN go mod download
COPY media-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o media-service ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/media-service/media-service .
EXPOSE 50053
CMD ["./media-service"]