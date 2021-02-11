FROM golang:latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go run cmd/start/start.go
EXPOSE 8080
