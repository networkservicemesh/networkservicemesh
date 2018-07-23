FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN apk add --update gcc
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /go/bin/nsm-init ./cmd/nsm-init/nsm-init.go

FROM alpine as runtime
COPY --from=build /go/bin/nsm-init /go/bin/nsm-init
ENTRYPOINT ["/go/bin/nsm-init"]
