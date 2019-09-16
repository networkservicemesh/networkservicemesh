package tools

import (
	"context"
	"fmt"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"testing"
)

type testMsg struct {
	str string
}

func (*testMsg) ProtoMessage() {
}

func (*testMsg) Reset() {
}

func (t *testMsg) String() string {
	return ""
}

func ptrToStr(t interface{}) string {
	return fmt.Sprintf("%p", t)
}

func TestCloneArgsClientInterceptor(t *testing.T) {
	g := NewWithT(t)

	globalReq := &testMsg{}
	globalResp := &testMsg{}

	const simpleMethod = "simple"
	const cloneMethod = "clone"

	ami := func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if method == simpleMethod {
			g.Expect(ptrToStr(req)).To(Equal(ptrToStr(globalReq)))
			g.Expect(ptrToStr(resp)).To(Equal(ptrToStr(globalResp)))
		}

		if method == cloneMethod {
			g.Expect(ptrToStr(req)).ToNot(Equal(ptrToStr(globalReq)))
			g.Expect(ptrToStr(resp)).ToNot(Equal(ptrToStr(globalResp)))
		}

		return nil
	}

	ami(context.Background(), simpleMethod, globalReq, globalResp, nil, nil)

	cloneArgsAmi := CloneArgsClientInterceptor(ami)
	cloneArgsAmi(context.Background(), cloneMethod, globalReq, globalResp, nil, nil)
}
