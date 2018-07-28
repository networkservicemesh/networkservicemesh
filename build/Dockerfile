FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
RUN apk add --update protobuf git bash gcc
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN ./scripts/build.sh --race-test-disabled
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o /go/bin/nse ./cmd/nse/nse.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o /go/bin/nsm-init ./cmd/nsm-init/nsm-init.go
