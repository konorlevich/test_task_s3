FROM golang:1.22.1 as golang

WORKDIR /usr/local/go/src/test_task_s3

COPY go.mod go.sum ./

RUN go mod download

FROM ubuntu as ubuntu

RUN apt update && apt install -y curl && rm -rf /var/lib/apt/lists/*

