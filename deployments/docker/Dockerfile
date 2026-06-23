FROM golang:1.25.11-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /bin/identity-api cmd/identity-api/main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /bin/identity-api /app/identity-api
CMD ["/app/identity-api"]
