#build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

#runtime
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY sql/schema ./sql/schema
COPY docker/entrypoint.sh .
RUN chmod +x entrypoint.sh
CMD ["./entrypoint.sh"]

