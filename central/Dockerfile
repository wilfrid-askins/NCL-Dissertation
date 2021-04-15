FROM golang:1.16.3-alpine3.13
# Cache dependency download
WORKDIR /app
COPY go.mod .
RUN go mod download

# Copy source code
COPY . .
# Build/run code
RUN go build cmd/start/start.go
EXPOSE 5000
ENTRYPOINT ["/app/start"]
