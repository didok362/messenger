FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/server /app/server
COPY web/ ./web/
COPY assets/ ./assets/
EXPOSE 8080
CMD ["/app/server"]

  