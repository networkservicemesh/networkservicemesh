FROM golang:1 as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/

RUN apt-get update && \
	apt-get install -y git build-essential autoconf pkg-config libtool sudo check

RUN git clone https://gerrit.fd.io/r/vpp /vpp
WORKDIR /vpp/extras/libmemif
RUN ./bootstrap && ./configure && make install
RUN ulimit -c unlimited

COPY [".","/go/src/${PACKAGEPATH}"]
WORKDIR /go/src/${PACKAGEPATH}/
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-extldflags "-static"' -o /go/bin/nse ./controlplane/cmd/nse/nse.go
FROM alpine as runtime
COPY --from=build /go/bin/nse /bin/nse
ENTRYPOINT ["/bin/nse"]
