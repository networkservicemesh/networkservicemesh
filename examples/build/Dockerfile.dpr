FROM golang:alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o /go/bin/dpr ./examples/cmd/dpr/dpr.go
FROM alpine as runtime
COPY --from=build /go/bin/dpr /bin/dpr
ENTRYPOINT ["/bin/dpr"]
