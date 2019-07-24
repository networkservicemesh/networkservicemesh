package tools

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	opentracingEnv     = "OPEN_TRACING"
	opentracingDefault = true
	insecureEnv        = "INSECURE"
	insecureDefault    = true
	dialTimeoutDefault = 5 * time.Second
)

type dialConfig struct {
	opentracing bool
	insecure    bool
}

var cfg dialConfig

func init() {
	var err error
	cfg, err = readConfiguration()
	if err != nil {
		logrus.Fatal(err)
	}
}

func readConfiguration() (dialConfig, error) {
	rv := dialConfig{}

	if ot, err := readEnvBool(opentracingEnv, opentracingDefault); err == nil {
		rv.opentracing = ot
	} else {
		return dialConfig{}, err
	}

	if insecure, err := readEnvBool(insecureEnv, insecureDefault); err == nil {
		rv.insecure = insecure
	} else {
		return dialConfig{}, err
	}

	return rv, nil
}

func readEnvBool(env string, value bool) (bool, error) {
	str := os.Getenv(env)
	if str == "" {
		return value, nil
	}

	return strconv.ParseBool(str)
}

// DialContext checks dialConfig and calls grpc.DialContext with certain grpc.DialOption
func DialContext(ctx context.Context, addr net.Addr, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if cfg.insecure {
		opts = append(opts, grpc.WithInsecure())
	}

	if cfg.opentracing {
		opts = append(opts,
			grpc.WithUnaryInterceptor(
				otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer(), otgrpc.LogPayloads())),
			grpc.WithStreamInterceptor(
				otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())))
	}

	opts = append(opts,
		grpc.WithBlock(),
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return net.Dial(addr.Network(), target)
		}))

	return grpc.DialContext(ctx, addr.String(), opts...)
}

// DialTimeout tries to establish connection with addr during timeout
func DialTimeout(addr net.Addr, timeout time.Duration, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return DialContext(ctx, addr, opts...)
}

// DialContextUnix establish connection with passed unix socket
func DialContextUnix(ctx context.Context, path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, err
	}
	return DialContext(ctx, addr, opts...)
}

// DialTimeoutUnix tries to establish connection with passed unix socket during timeout
func DialTimeoutUnix(path string, timeout time.Duration, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return DialContextUnix(ctx, path, opts...)
}

// DialUnix simply calls DialTimeoutUnix with default timeout
func DialUnix(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialTimeoutUnix(path, dialTimeoutDefault, opts...)
}

// DialContextTCP establish TCP connection with address
func DialContextTCP(ctx context.Context, address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	return DialContext(ctx, addr, opts...)
}

// DialTimeoutTCP tries to establish connection with passed address during timeout
func DialTimeoutTCP(address string, timeout time.Duration, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return DialContextTCP(ctx, address, opts...)
}

// DialTCP simply calls DialTimeoutTCP with default timeout
func DialTCP(address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialTimeoutTCP(address, dialTimeoutDefault, opts...)
}
