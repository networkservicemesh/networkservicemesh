FROM golang:alpine as build
RUN apk add --update protobuf git bash gcc musl-dev curl
ENV PACKAGEPATH=github.com/networkservicemesh/networkservicemesh/
ENV GO111MODULE=on

RUN mkdir /root/networkservicemesh
ADD ["go.mod","/root/networkservicemesh"]
ADD ["./scripts/go-mod-download.sh","/root/networkservicemesh"]
WORKDIR /root/networkservicemesh/
RUN ./go-mod-download.sh

#RUN go get -u github.com/golang/protobuf/protoc-gen-go
#RUN go get -u golang.org/x/tools/cmd/stringer
#RUN go get -u k8s.io/code-generator/cmd/deepcopy-gen

ADD [".","/root/networkservicemesh"]
ENV GOCACHE=/go-cache
RUN go env GOCACHE
#RUN go generate ./...
#RUN CGO_ENABLED=0 GOOS=linux time go build -i -work -ldflags '-extldflags "-static"' ./...
