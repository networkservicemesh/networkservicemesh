FROM alpine as runtime
COPY --from=networkservicemesh/release /go/bin/nse /go/bin/nse
ENTRYPOINT ["/go/bin/nse"]
