FROM golang:1.10-alpine3.7 as build
ENV PACKAGEPATH=github.com/ligato/networkservicemesh/
RUN apk add --update protobuf git
COPY [".","/go/src/${PACKAGEPATH}"]
RUN test -z $(go fmt ./...) || (echo "Run go fmt ./... and recommit your code";exit 1)
WORKDIR /go/src/${PACKAGEPATH}/
RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN PATH=${PATH}:/go/bin go generate ./...
RUN go build ./...
RUN go test ./...
RUN go install ./...

FROM alpine:3.7 as runtime
COPY --from=build /go/bin/nsm /go/bin/nsm
ENTRYPOINT ["/go/bin/nsm"]


