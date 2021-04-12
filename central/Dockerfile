FROM golang:latest

WORKDIR /app

COPY . .

WORKDIR /app
RUN go mod download
RUN go build cmd/start/start.go

EXPOSE 5000
ENTRYPOINT ["/app/start"]