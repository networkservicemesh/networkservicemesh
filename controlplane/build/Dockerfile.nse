FROM golang:alpine as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o /go/bin/nse ./controlplane/cmd/nse/nse.go
FROM alpine as runtime
COPY --from=build /go/bin/nse /bin/nse
ENTRYPOINT ["/bin/nse"]
