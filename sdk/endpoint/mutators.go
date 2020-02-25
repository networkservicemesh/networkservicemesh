package endpoint

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

//CreateRouteMutator - Creates an instance of ConnectionMutator with routes mutating
func CreateRouteMutator(routes []string) ConnectionMutator {
	return func(ctc context.Context, c *networkservice.Connection) error {
		for _, r := range routes {
			c.GetContext().GetIpContext().DstRoutes = append(c.GetContext().GetIpContext().GetDstRoutes(), &networkservice.Route{
				Prefix: r,
			})
		}
		return nil
	}
}

func CreatePodNameMutator() ConnectionMutator {
	return func(ctc context.Context, c *networkservice.Connection) error {
		podName, err := tools.GetCurrentPodNameFromHostname()
		if err != nil {
			logrus.Infof("failed to get current pod name from hostname: %v", err)
		} else {
			c.Labels[networkservice.PodNameKey] = podName
			c.Labels[networkservice.NamespaceKey] = common.GetNamespace()
		}
		return nil
	}

}
