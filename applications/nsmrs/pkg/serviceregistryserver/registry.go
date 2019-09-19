package serviceregistryserver

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/sirupsen/logrus"
	"time"
)

type nseRegistryService struct {
	//cache   RegistryCache
}

func NewNseRegistryService() *nseRegistryService {
	return &nseRegistryService{}
}

func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)

	return nil, fmt.Errorf("not implemented")
}


func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	st := time.Now()

	logrus.Infof("Received RemoveNSE(%v)", request)


	logrus.Infof("RemoveNSE done: time %v", time.Since(st))
	return &empty.Empty{}, nil
}

