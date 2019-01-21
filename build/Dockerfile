FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/networkservicemesh/networkservicemesh/
RUN apk add --update protobuf git bash gcc musl-dev
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN ./scripts/build.sh --race-test-disabled
