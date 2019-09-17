package tools

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/networkservicemesh/security"
)

const (
	// InsecureEnv environment variable, if "true" NSM will work in insecure mode
	InsecureEnv = "INSECURE"

	opentracingEnv     = "OPEN_TRACING"
	opentracingDefault = true
	insecureDefault    = false
	dialTimeoutDefault = 5 * time.Second
)

// DialConfig represents configuration of grpc connection, one per instance
type DialConfig struct {
	OpenTracing     bool
	SecurityManager security.Manager
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
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	if GetConfig().SecurityManager != nil {
		cred := credentials.NewTLS(&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{*GetConfig().SecurityManager.GetCertificate()},
			ClientCAs:    GetConfig().SecurityManager.GetCABundle(),
		})
		opts = append(opts, grpc.Creds(cred))
	}

	if GetConfig().OpenTracing {
		logrus.Infof("GRPC.NewServer with open tracing enabled")
		opts = append(opts,
			grpc.UnaryInterceptor(
				CloneArgsServerInterceptor(
					otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer(), otgrpc.LogPayloads()))),
			grpc.StreamInterceptor(
				otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())))
	}

	return grpc.NewServer(opts...)
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

type dialContextFunc func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)

type dialBuilder struct {
	opts []grpc.DialOption
	t    time.Duration
}

func (b *dialBuilder) TCP() *dialBuilder {
	return b.Network("tcp")
}

func (b *dialBuilder) Unix() *dialBuilder {
	return b.Network("unix")
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
		if b.t != 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, b.t)
			defer cancel()
		}

		if GetConfig().OpenTracing {
			b.opts = append(b.opts, OpenTracingDialOptions()...)
		}

		if GetConfig().SecurityManager != nil {
			cred := credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{*GetConfig().SecurityManager.GetCertificate()},
				RootCAs:            GetConfig().SecurityManager.GetCABundle(),
			})
			opts = append(opts, grpc.WithTransportCredentials(cred))
		} else {
			opts = append(opts, grpc.WithInsecure())
		}

		b.opts = append(b.opts, grpc.WithBlock())

		return grpc.DialContext(ctx, target, append(opts, b.opts...)...)
	}
}

// OpenTracingDialOptions returns array of grpc.DialOption that should be passed to grpc.Dial to enable opentracing
func OpenTracingDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(
			CloneArgsClientInterceptor(
				otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer(), otgrpc.LogPayloads()))),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())),
	}
}

func readDialConfig() (DialConfig, error) {
	rv := DialConfig{}

	if ot, err := ReadEnvBool(opentracingEnv, opentracingDefault); err == nil {
		rv.OpenTracing = ot
	} else {
		return DialConfig{}, err
	}

	insecure, err := IsInsecure()
	if err != nil {
		return DialConfig{}, err
	}

	if !insecure {
		rv.SecurityManager = security.NewManager()
	}

	return rv, nil
}
