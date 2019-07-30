package fanout

import (
	"context"
	"testing"
	"time"

	"github.com/coredns/coredns/request"
)

func TestNoDnstap(t *testing.T) {
	err := toDnstap(context.TODO(), "", request.Request{}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}
