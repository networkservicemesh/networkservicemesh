package tools

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"
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
func GetConfig() *DialConfig {
	once.Do(func() {
		var err error
		cfg, err = readDialConfig()
		if err != nil {
			logrus.Fatal(err)
		}
	})
	return &cfg
}

// InitConfig allows init global DialConfig, should be called before any GetConfig(), otherwise do nothing
func InitConfig(c DialConfig) {
	once.Do(func() {
		cfg = c
	})
}

// NewServer checks DialConfig and calls grpc.NewServer with certain grpc.ServerOption
func NewServer(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server {
	newServer := new(NewServerBuilder).NewServerFunc()
	return newServer(ctx, opts...)
}

func NewServerWithToken(ctx context.Context, cfg security.TokenConfig, opts ...grpc.ServerOption) *grpc.Server {
	newServer := new(NewServerBuilder).TokenVerification(cfg).NewServerFunc()
	return newServer(ctx, opts...)
}

// NewServerInsecure calls grpc.NewServer without security even if it specified in DialConfig
func NewServerInsecure(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server {
	newServer := new(NewServerBuilder).Insecure().NewServerFunc()
	return newServer(ctx, opts...)
}

// DialContext allows to call DialContext using net.Addr
func DialContext(ctx context.Context, addr net.Addr, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).Network(addr.Network()).DialContextFunc()
	return dialCtx(ctx, addr.String(), opts...)
}

// DialUnix establish connection with passed unix socket
func DialUnix(ctx context.Context, path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).Unix().DialContextFunc()
	return dialCtx(ctx, path, opts...)
}

func DialUnixWithToken(ctx context.Context, address string, cfg security.TokenConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).Unix().TokenVerification(cfg).DialContextFunc()
	return dialCtx(ctx, address, opts...)
}

// DialUnixInsecure establish connection with passed unix socket in insecure mode and set default timeout
func DialUnixInsecure(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).Unix().Insecure().Timeout(dialTimeoutDefault).DialContextFunc()
	return dialCtx(context.Background(), path, opts...)
}

// DialTCP establish TCP connection with address
func DialTCP(ctx context.Context, address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).TCP().DialContextFunc()
	return dialCtx(ctx, address, opts...)
}

func DialTCPWithToken(ctx context.Context, address string, cfg security.TokenConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).TCP().TokenVerification(cfg).DialContextFunc()
	return dialCtx(ctx, address, opts...)
}

// DialTCPInsecure establish TCP connection with address in insecure mode and set default timeout
func DialTCPInsecure(address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialCtx := new(DialBuilder).TCP().Insecure().Timeout(dialTimeoutDefault).DialContextFunc()
	return dialCtx(context.Background(), address, opts...)
}

type DialContextFunc func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)

type DialBuilder struct {
	opts     []grpc.DialOption
	t        time.Duration
	insecure bool
	cfg      *DialConfig
	token    security.TokenConfig
}

func (b *DialBuilder) TCP() *DialBuilder {
	return b.Network("tcp")
}

func (b *DialBuilder) Unix() *DialBuilder {
	return b.Network("unix")
}

func (b *DialBuilder) Insecure() *DialBuilder {
	b.insecure = true
	return b
}

func (b *DialBuilder) TokenVerification(cfg security.TokenConfig) *DialBuilder {
	b.token = cfg
	return b
}

func (b *DialBuilder) Network(network string) *DialBuilder {
	b.opts = append(b.opts, grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, target)
	}))
	return b
}

func (b *DialBuilder) Timeout(t time.Duration) *DialBuilder {
	b.t = t
	return b
}

func (b *DialBuilder) WithConfig(cfg *DialConfig) *DialBuilder {
	b.cfg = cfg
	return b
}

func (b *DialBuilder) DialContextFunc() DialContextFunc {
	return func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
		if b.cfg == nil {
			// config doesn't set explicitly, use global config
			b.cfg = GetConfig()
		}

		unaryInts := []grpc.UnaryClientInterceptor{}

		if b.cfg.OpenTracing {
			b.opts = append(b.opts,
				grpc.WithStreamInterceptor(
					otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())))

			unaryInts = append(unaryInts,
				CloneArgsClientInterceptor(
					otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer())))
		}

		if !b.insecure && b.cfg.SecurityProvider != nil {
			cred := credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{*b.cfg.SecurityProvider.GetCertificate()},
				RootCAs:            b.cfg.SecurityProvider.GetCABundle(),
			})
			b.opts = append(b.opts, grpc.WithTransportCredentials(cred))
			//unaryInts = append(unaryInts, security.ClientInterceptor(b.cfg.SecurityProvider))
		} else {
			b.opts = append(b.opts, grpc.WithInsecure())
		}

		if !b.insecure && b.cfg.SecurityProvider != nil && b.token != nil {
			unaryInts = append(unaryInts, security.ClientInterceptor(b.cfg.SecurityProvider, b.token))
		}

		b.opts = append(b.opts, grpc.WithBlock())
		b.opts = append(b.opts, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(unaryInts...)))

		if b.t != 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, b.t)
			defer cancel()
		}

		return grpc.DialContext(ctx, target, append(opts, b.opts...)...)
	}
}

type NewServerFunc func(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server

type NewServerBuilder struct {
	insecure bool
	cfg      *DialConfig
	token    security.TokenConfig
}

func (b *NewServerBuilder) Insecure() *NewServerBuilder {
	b.insecure = true
	return b
}

func (b *NewServerBuilder) WithConfig(cfg *DialConfig) *NewServerBuilder {
	b.cfg = cfg
	return b
}

func (b *NewServerBuilder) TokenVerification(cfg security.TokenConfig) *NewServerBuilder {
	b.token = cfg
	return b
}

func (b *NewServerBuilder) NewServerFunc() NewServerFunc {
	return func(ctx context.Context, opts ...grpc.ServerOption) *grpc.Server {
		span := spanhelper.FromContext(ctx, "NewServer")
		defer span.Finish()

		var unaryInts []grpc.UnaryServerInterceptor
		if b.cfg == nil {
			// config doesn't set explicitly, use global config
			b.cfg = GetConfig()
		}

		if !b.insecure && b.cfg.SecurityProvider != nil {
			securitySpan := spanhelper.FromContext(span.Context(), "GetCertificate")
			certificate := b.cfg.SecurityProvider.GetCertificate()
			cred := credentials.NewTLS(&tls.Config{
				ClientAuth:   tls.RequireAndVerifyClientCert,
				Certificates: []tls.Certificate{*certificate},
				ClientCAs:    b.cfg.SecurityProvider.GetCABundle(),
			})
			opts = append(opts, grpc.Creds(cred))
			securitySpan.Finish()
		}

		if !b.insecure && b.cfg.SecurityProvider != nil && b.token != nil {
			unaryInts = append(unaryInts, security.ServerInterceptor(b.cfg.SecurityProvider, b.token))
		}

		if b.cfg.OpenTracing {
			span.Logger().Infof("GRPC.NewServer with open tracing enabled")
			opts = append(opts,
				grpc.StreamInterceptor(
					otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer())))

			unaryInts = append(unaryInts,
				CloneArgsServerInterceptor(
					otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())))
		}

		opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInts...)))
		return grpc.NewServer(opts...)
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
		rv.SecurityProvider = security.NewProvider()
	}

	return rv, nil
}
