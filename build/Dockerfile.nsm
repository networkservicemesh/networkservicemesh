FROM alpine
COPY --from=networkservicemesh/release /go/bin/netmesh /go/bin/netmesh
RUN mkdir -p /var/lib/kubelet/device-plugins
ENV GOTRACEBACK=system
ENTRYPOINT ["/go/bin/netmesh"]
