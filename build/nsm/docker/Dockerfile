FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
RUN apk add --update protobuf git bash
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN ./scripts/build.sh --race-test-disabled

FROM alpine as runtime
COPY --from=build /go/bin/nsm /go/bin/nsm
ENTRYPOINT ["/go/bin/nsm"]
