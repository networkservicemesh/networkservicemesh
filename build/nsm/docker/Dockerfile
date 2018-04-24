FROM golang:1.10-alpine3.7 as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
RUN apk add --update protobuf git bash
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN ./scripts/build.sh

FROM alpine:3.7 as runtime
COPY --from=build /go/bin/nsm /go/bin/nsm
ENTRYPOINT ["/go/bin/nsm"]
