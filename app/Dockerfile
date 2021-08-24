# This Dockerfile containsanti-patterns!

FROM golang:1.16-buster

LABEL org.opencontainers.image.source https://github.com/nmeisenzahl/hijack-kubernetes

WORKDIR /app

ENV CGO_ENABLED=0
ADD . .
RUN go build -o sample-app

EXPOSE 3000

ENTRYPOINT ["./sample-app"]
