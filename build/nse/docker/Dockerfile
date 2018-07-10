FROM golang:1-alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /go/bin/nse ./cmd/nse/nse.go

FROM alpine as runtime
COPY --from=build /go/bin/nse /go/bin/nse
ENTRYPOINT ["/go/bin/nse"]
