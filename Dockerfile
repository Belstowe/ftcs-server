FROM golang:1.20-alpine AS build-stage
RUN mkdir -p /server
WORKDIR /server
COPY . .
RUN go build -o ftcs-server .

FROM alpine:3.17
COPY --from=build-stage /server/ftcs-server /usr/local/bin/ftcs-server
CMD /usr/local/bin/ftcs-server
