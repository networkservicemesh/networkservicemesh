FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN CGO_ENABLED=0 GOOS=linux go build -a -v -ldflags '-extldflags "-static"' -o /go/bin/sidecar-injector ./cmd/sidecar-injector/main.go

FROM alpine as runtime
COPY --from=build /go/bin/sidecar-injector /go/bin/sidecar-injector
ENTRYPOINT ["/go/bin/sidecar-injector"]
