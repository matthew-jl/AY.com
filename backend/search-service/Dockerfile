FROM golang:1.23 AS builder 
WORKDIR /app/search-service 
COPY search-service/go.mod search-service/go.sum ./ 
RUN go mod download 
COPY search-service/ . 
RUN CGO_ENABLED=0 GOOS=linux go build -o search-service ./cmd

FROM alpine:latest 
WORKDIR /root/ 
COPY --from=builder /app/search-service/search-service . 
EXPOSE 50054 
CMD ["./search-service"]