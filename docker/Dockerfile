FROM golang:1.18.4

ARG BUILDTIME_VARIABLE="default value"
ARG BUILDTIME_VARIABLE_TWO="default value"

ADD . /server
WORKDIR /server
RUN go build
RUN ./capture-build-envs.sh

CMD [ "/server/go-info-webserver" ]