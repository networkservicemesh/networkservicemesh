package main

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/k8s/api/nsm-coredns/update"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/utils"
)

type updateServer struct {
	dnsConfigManager        *utils.DNSConfigManager
	corefilePath            string
	updateCorefileOperation utils.Operation
}

func (s *updateServer) AddDNSContext(ctx context.Context, msg *update.AddDNSContextMessage) (*empty.Empty, error) {
	for _, c := range msg.Context.Configs {
		fmt.Printf("Added new config %v with connection id %v", c, msg.ConnectionID)
		s.dnsConfigManager.Store(msg.ConnectionID, *c)
	}
	s.updateCorefileOperation.Run()
	return new(empty.Empty), nil
}

func (s *updateServer) RemoveDNSContext(ctx context.Context, msg *update.RemoveDNSContextMessage) (*empty.Empty, error) {
	s.dnsConfigManager.Delete(msg.ConnectionID)
	s.updateCorefileOperation.Run()
	return new(empty.Empty), nil
}
