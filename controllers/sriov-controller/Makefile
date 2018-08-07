REGISTRY_NAME = 192.168.80.240:4000/ligato
IMAGE_VERSION = latest

.PHONY: all sriov-controller nsm-generate-sriov-configmap mac-controller mac-sriov-configmap container push clean test

ifdef V
TESTARGS = -v -args -alsologtostderr -v 5
else
TESTARGS =
endif

all: sriov-controller nsm-generate-sriov-configmap

sriov-controller:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o ./bin/nsm-generate-sriov-configmap ./sriov-controller.go service-controller.go dpapi-controller.go

nsm-generate-sriov-configmap:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o ./bin/nsm-generate-sriov-configmap ./nsm-generate-sriov-configmap/nsm-generate-sriov-configmap.go

mac-controller:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin go build -a -ldflags '-extldflags "-static"' -o ./bin/sriov-controller.mac ./sriov-controller.go service-controller.go dpapi-controller.go

mac-sriov-configmap:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin go build -a -ldflags '-extldflags "-static"' -o ./bin/nsm-generate-sriov-configmap.mac ./nsm-generate-sriov-configmap/nsm-generate-sriov-configmap.go

container: sriov-controller
	docker build -t $(REGISTRY_NAME)/sriov-controller:$(IMAGE_VERSION) -f ./Dockerfile .

push: container
	docker push $(REGISTRY_NAME)/sriov-controller:$(IMAGE_VERSION)

clean:
	rm -rf bin

test:
	go test `go list ./... | grep -v 'vendor'` $(TESTARGS)
	go vet `go list ./... | grep -v vendor`
