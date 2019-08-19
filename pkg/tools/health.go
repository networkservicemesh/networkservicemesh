package tools

import (
	"context"
	"errors"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func CheckHealth(addr net.Addr, timeout time.Duration, opts ...grpc.DialOption) error {
	ctx, close := context.WithTimeout(context.Background(), timeout)
	defer close()
	conn, err := DialContext(ctx, addr, opts...)
	if err != nil {
		return err
	}

	resp, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})

	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return errors.New("service is not serving")
	}
	return nil
}
