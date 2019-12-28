package vppagent

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/converter"
)

//KernelInterfaces creates forwarder server handler with creation dataChange config for kernel and not direct memif connections
func KernelInterfaces(baseDir string) forwarder.ForwarderServer {
	return &kernelInterfaces{baseDir: baseDir}
}

type kernelInterfaces struct {
	baseDir string
}

func (c *kernelInterfaces) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	conversionParameters := &converter.CrossConnectConversionParameters{
		BaseDir: c.baseDir,
	}
	dataChange, err := converter.NewCrossConnectConverter(crossConnect, conversionParameters).ToDataRequest(nil, true)
	if err != nil {
		return nil, err
	}
	nextCtx := WithDataChange(ctx, dataChange)
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(nextCtx, crossConnect)
}

func (c *kernelInterfaces) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	conversionParameters := &converter.CrossConnectConversionParameters{
		BaseDir: c.baseDir,
	}
	dataChange, err := converter.NewCrossConnectConverter(crossConnect, conversionParameters).ToDataRequest(nil, false)
	if err != nil {
		return nil, err
	}
	nextCtx := WithDataChange(ctx, dataChange)
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(nextCtx, crossConnect)
}
