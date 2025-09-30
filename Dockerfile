FROM golang:1.24-alpine

WORKDIR /app

# Cache modules
COPY go.mod go.sum ./

# Copy source and build
RUN apk add --no-cache ca-certificates tzdata docker-cli
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o /app/app ./cmd/api

ENV GIN_MODE=release \
    PORT=8080

EXPOSE 8080
CMD ["/app/app"]
