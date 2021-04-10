FROM golang:latest

WORKDIR /app

COPY ./central ./central
COPY ./netpb ./netpb

WORKDIR /app/central
RUN go mod download
RUN go build cmd/start/start.go

EXPOSE 5000
ENTRYPOINT ["/app/central/start"]