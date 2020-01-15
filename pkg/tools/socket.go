package tools

import (
	"context"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// InsecureEnv environment variable, if "true" NSM will work in insecure mode
	InsecureEnv = "INSECURE"

	insecureDefault    = false
	dialTimeoutDefault = 15 * time.Second
)

// DialConfig represents configuration of grpc connection, one per instance
type DialConfig struct {
	OpenTracing      bool
	SecurityProvider security.Provider
}

var cfg DialConfig
var once sync.Once

// GetConfig returns instance of DialConfig
func GetConfig() DialConfig {
	once.Do(func() {
		var err error
		cfg, err = readDialConfig()
		if err != nil {
			logrus.Fatal(err)
		}
	})
	return cfg
}

// InitConfig allows init global DialConfig, should be called before any GetConfig(), otherwise do nothing
func InitConfig(c DialConfig) {
	once.Do(func() {
		cfg = c
	})
}

// NewServer checks DialConfig and calls grpc.NewServer with certain grpc.ServerOption
func NewServer(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server {
	span := spanhelper.FromContext(ctx, "NewServer")
	defer span.Finish()
	if GetConfig().SecurityProvider != nil {
		securitySpan := spanhelper.FromContext(span.Context(), "GetCertificate")
		tlscfg, err := GetConfig().SecurityProvider.GetTLSConfig(ctx)
		if err != nil {
			return nil
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlscfg)))
		securitySpan.Finish()
	}

	if GetConfig().OpenTracing {
		span.Logger().Infof("GRPC.NewServer with open tracing enabled")
		opts = append(opts, openTracingOpts()...)
	}

	return grpc.NewServer(opts...)
}

func NewServerInsecure(opts ...grpc.ServerOption) *grpc.Server {
	if GetConfig().OpenTracing {
		logrus.Infof("GRPC.NewServer with open tracing enabled")
		opts = append(opts, openTracingOpts()...)
	}

	return grpc.NewServer(opts...)
}

func openTracingOpts() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(
			CloneArgsServerInterceptor(
				otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer()))),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())),
	}
}

// DialContext allows to call DialContext using net.Addr
func DialContext(ctx context.Context, addr net.Addr, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).Network(addr.Network()).DialContextFunc()
	return dialCtx(ctx, addr.String(), opts...)
}

// DialContextUnix establish connection with passed unix socket
func DialContextUnix(ctx context.Context, path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).Unix().DialContextFunc()
	return dialCtx(ctx, path, opts...)
}

// DialUnix establish connection with passed unix socket and set default timeout
func DialUnix(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).Unix().Timeout(dialTimeoutDefault).DialContextFunc()
	return dialCtx(context.Background(), path, opts...)
}

// DialUnixInsecure establish connection with passed unix socket in insecure mode and set default timeout
func DialUnixInsecure(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).Unix().Insecure().Timeout(dialTimeoutDefault).DialContextFunc()
	return dialCtx(context.Background(), path, opts...)
}

// DialContextTCP establish TCP connection with address
func DialContextTCP(ctx context.Context, address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).TCP().DialContextFunc()
	return dialCtx(ctx, address, opts...)
}

// DialTCP establish TCP connection with address and set default timeout
func DialTCP(address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).TCP().Timeout(dialTimeoutDefault).DialContextFunc()
	return dialCtx(context.Background(), address, opts...)
}

// DialTCPInsecure establish TCP connection with address in insecure mode and set default timeout
func DialTCPInsecure(address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(dialBuilder).TCP().Insecure().Timeout(dialTimeoutDefault).DialContextFunc()
	return dialCtx(context.Background(), address, opts...)
}

func DialUrl(u *url.URL, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if u.Scheme == "unix" {
		return DialUnix(u.Path, opts...)
	}
	return DialTCP(u.Host, opts...)
}

type dialContextFunc func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)

type dialBuilder struct {
	opts     []grpc.DialOption
	t        time.Duration
	insecure bool
}

func (b *dialBuilder) TCP() *dialBuilder {
	return b.Network("tcp")
}

func (b *dialBuilder) Unix() *dialBuilder {
	return b.Network("unix")
}

func (b *dialBuilder) Insecure() *dialBuilder {
	b.insecure = true
	return b
}

func (b *dialBuilder) Network(network string) *dialBuilder {
	b.opts = append(b.opts, grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, target)
	}))
	return b
}

func (b *dialBuilder) Timeout(t time.Duration) *dialBuilder {
	b.t = t
	return b
}

func (b *dialBuilder) DialContextFunc() dialContextFunc {
	return func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
		if GetConfig().OpenTracing {
			b.opts = append(b.opts, OpenTracingDialOptions()...)
		}

		if !b.insecure && GetConfig().SecurityProvider != nil {
			tlscfg, err := GetConfig().SecurityProvider.GetTLSConfig(ctx)
			if err != nil {
				return nil, err
			}
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlscfg)))
		} else {
			opts = append(opts, grpc.WithInsecure())
		}

		b.opts = append(b.opts, grpc.WithBlock())

		if b.t != 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, b.t)
			defer cancel()
		}

		return grpc.DialContext(ctx, target, append(opts, b.opts...)...)
	}
}

// OpenTracingDialOptions returns array of grpc.DialOption that should be passed to grpc.Dial to enable opentracing
func OpenTracingDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(
			CloneArgsClientInterceptor(
				otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()))),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())),
	}
}

func readDialConfig() (DialConfig, error) {
	rv := DialConfig{
		OpenTracing: jaeger.IsOpentracingEnabled(),
	}

	insecure, err := IsInsecure()
	if err != nil {
		return DialConfig{}, err
	}

	if !insecure {
		rv.SecurityProvider, err = security.NewSpireProvider(security.SpireAgentUnixAddr)
		if err != nil {
			return DialConfig{}, err
		}
	}

	return rv, nil
}
