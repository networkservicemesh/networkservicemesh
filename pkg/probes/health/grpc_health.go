package health

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

//NewGrpcHealth creates health checker for grpc servers
func NewGrpcHealth(s *grpc.Server, addr net.Addr, timeout time.Duration, opts ...grpc.DialOption) ApplicationHealth {
	grpc_health_v1.RegisterHealthServer(s, &healhServiceImpl{})
	return NewApplicationHealthFunc(
		func() error {
			ctx, closeFn := context.WithTimeout(context.Background(), timeout)
			defer closeFn()

			conn, err := tools.DialContext(ctx, addr, opts...)
			defer func() { _ = conn.Close() }()

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
		})
}
