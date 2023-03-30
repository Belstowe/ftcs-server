FROM golang:1.20-alpine
RUN mkdir -p /server
WORKDIR /server
COPY . .
CMD go run main.go
