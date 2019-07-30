package prefixcollector

import (
	"context"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
)

type prefixService struct {
	sync.RWMutex
	excludedPrefixes prefix_pool.PrefixPool
}

func newPrefixService(config *rest.Config) (plugins.ConnectionPluginServer, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	emptyPrefixPool, err := prefix_pool.NewPrefixPool()
	if err != nil {
		return nil, err
	}

	rv := &prefixService{
		excludedPrefixes: emptyPrefixPool,
	}

	if err := rv.monitorExcludedPrefixes(clientset); err != nil {
		return nil, err
	}

	return rv, nil
}

func (c *prefixService) getExcludedPrefixes() prefix_pool.PrefixPool {
	c.RLock()
	defer c.RUnlock()

	return c.excludedPrefixes
}

func (c *prefixService) setExcludedPrefixes(prefixes prefix_pool.PrefixPool) {
	c.Lock()
	defer c.Unlock()

	c.excludedPrefixes = prefixes
}

func (c *prefixService) monitorExcludedPrefixes(clientset *kubernetes.Clientset) error {
	poolCh, err := getExcludedPrefixesChan(clientset)
	if err != nil {
		return err
	}

	go func() {
		for pool := range poolCh {
			c.setExcludedPrefixes(pool)
		}
	}()

	return nil
}

func (c *prefixService) UpdateConnectionContext(ctx context.Context, connCtx *connectioncontext.ConnectionContext) (*connectioncontext.ConnectionContext, error) {
	connCtx.GetIpContext().ExcludedPrefixes = append(connCtx.GetIpContext().GetExcludedPrefixes(), c.getExcludedPrefixes().GetPrefixes()...)
	return connCtx, nil
}

func (c *prefixService) ValidateConnectionContext(ctx context.Context, connCtx *connectioncontext.ConnectionContext) (*plugins.ConnectionValidationResult, error) {
	prefixes := c.getExcludedPrefixes()

	if srcIP := connCtx.GetIpContext().GetSrcIpAddr(); srcIP != "" {
		intersect, err := prefixes.Intersect(srcIP)
		if err != nil {
			return nil, err
		}
		if intersect {
			return &plugins.ConnectionValidationResult{
				Status:       plugins.ConnectionValidationStatus_FAIL,
				ErrorMessage: "srcIP intersects excluded prefixes list",
			}, nil
		}
	}

	if dstIP := connCtx.GetIpContext().GetDstIpAddr(); dstIP != "" {
		intersect, err := prefixes.Intersect(dstIP)
		if err != nil {
			return nil, err
		}
		if intersect {
			return &plugins.ConnectionValidationResult{
				Status:       plugins.ConnectionValidationStatus_FAIL,
				ErrorMessage: "dstIP intersects excluded prefixes list",
			}, nil
		}
	}

	return &plugins.ConnectionValidationResult{
		Status: plugins.ConnectionValidationStatus_SUCCESS,
	}, nil
}
