# syntax=docker/dockerfile:1

FROM golang:1.18

WORKDIR /app

COPY . ./

ENV GOFLAGS=-mod=vendor
RUN go build -o random-string-generator generator.go

EXPOSE 8080

CMD ./random-string-generator
