FROM golang:1.14.2-buster

ADD . /server
WORKDIR /server
RUN go build
RUN ./capture-build-envs.sh

CMD [ "/server/go-info-webserver" ]