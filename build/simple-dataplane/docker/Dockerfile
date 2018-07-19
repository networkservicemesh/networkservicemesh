FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /go/bin/simple-dataplane ./cmd/simple-dataplane/simple-dataplane.go

FROM alpine as runtime
COPY --from=build /go/bin/simple-dataplane /go/bin/simple-dataplane
ENTRYPOINT ["/go/bin/simple-dataplane"]
