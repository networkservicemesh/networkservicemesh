FROM alpine as runtime
COPY --from=networkservicemesh/release /go/bin/nsm-init /go/bin/nsm-init
ENTRYPOINT ["/go/bin/nsm-init"]
