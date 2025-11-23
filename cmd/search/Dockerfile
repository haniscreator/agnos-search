# builder
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/search ./cmd/search

# final
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/search /bin/search
EXPOSE 8080
ENTRYPOINT ["/bin/search"]
