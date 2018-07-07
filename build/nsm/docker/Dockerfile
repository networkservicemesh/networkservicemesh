FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
RUN apk add --update protobuf git bash
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN ./scripts/build.sh --race-test-disabled

FROM alpine as runtime
COPY --from=build /go/bin/netmesh /go/bin/netmesh
RUN mkdir -p /var/lib/kubelet/device-plugins
ENTRYPOINT ["/go/bin/netmesh", "--kube-config=/conf/kube.conf"]
