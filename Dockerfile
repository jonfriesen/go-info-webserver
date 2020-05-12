FROM golang:1.14.2-buster AS builder

ADD . /server
WORKDIR /server
RUN go build
RUN ./capture-build-envs.sh
