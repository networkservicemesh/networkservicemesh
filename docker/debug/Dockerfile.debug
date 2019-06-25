FROM golang:alpine as build

RUN apk add --update protobuf git bash gcc musl-dev curl

# Compile Delve
ARG VERSION=unspecified
RUN echo "Building DLV"
RUN go get github.com/derekparker/delve/cmd/dlv

# Allow delve to run on Alpine based containers.
RUN apk add --no-cache libc6-compat

RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN go get -u golang.org/x/tools/cmd/stringer

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

ENV GO111MODULE=on
RUN mkdir /root/networkservicemesh
ADD ["go.mod","/root/networkservicemesh"]
WORKDIR /root/networkservicemesh/
RUN go mod download

COPY ["./docker/debug/dev-entry.go","/go/src/"]
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags "-static" -X  main.version=${VERSION}" -o /go/bin/dev-entry /go/src
ENTRYPOINT ["/go/bin/dev-entry"]