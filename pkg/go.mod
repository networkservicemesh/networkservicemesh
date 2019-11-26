module github.com/networkservicemesh/networkservicemesh/pkg

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.7.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spiffe/go-spiffe v0.0.0-20191104192205-d29ac0a1ba99
	github.com/spiffe/spire v0.0.0-20190515205011-c8123525fba8
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.17.0+incompatible
	github.com/uber/jaeger-lib v2.1.1+incompatible // indirect
	go.uber.org/atomic v1.4.0 // indirect
	google.golang.org/grpc v1.23.1
	gopkg.in/square/go-jose.v2 v2.3.0
)

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

go 1.13
