FROM alpine as runtime
COPY --from=networkservicemesh/release /go/bin/simple-dataplane /go/bin/simple-dataplane
ENTRYPOINT ["/go/bin/simple-dataplane"]
