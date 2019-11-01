package prefixcollector

import (
	"context"
	"sync"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
)

type prefixService struct {
	sync.RWMutex
	excludedPrefixes prefix_pool.PrefixPool
}

// NewPrefixService creates an instance of ConnectionPluginServer
func NewPrefixService(config *rest.Config) (plugins.ConnectionPluginServer, error) {
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

func (c *prefixService) UpdateConnection(ctx context.Context, wrapper *connection.Connection) (*connection.Connection, error) {
	connCtx := wrapper.GetContext()
	connCtx.GetIpContext().ExcludedPrefixes = append(connCtx.GetIpContext().GetExcludedPrefixes(), c.getExcludedPrefixes().GetPrefixes()...)
	return wrapper, nil
}

func (c *prefixService) ValidateConnection(ctx context.Context, wrapper *connection.Connection) (*plugins.ConnectionValidationResult, error) {
	prefixes := c.getExcludedPrefixes()
	connCtx := wrapper.GetContext()

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
