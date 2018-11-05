FROM alpine as runtime
COPY --from=networkservicemesh/release /go/bin/nsmdp /go/bin/nsmdp
ENTRYPOINT ["/go/bin/nsmdp"]
