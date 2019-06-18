package vppagent

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
)

const (
	kvSchedulerPort   = 9191
	kvSchedulerPrefix = "/scheduler"
	downStreamResync  = "/downstream-resync"
)

type kvSchedulerClient struct {
	httpClient          http.Client
	kvSchedulerEndpoint string
}

func newKVSchedulerClient(vppAgentEndpoint string) (*kvSchedulerClient, error) {
	kvSchedulerEndpoint, err := buildKvSchedulerDownStreamPath(vppAgentEndpoint)
	if err != nil {
		return nil, err
	}
	return &kvSchedulerClient{
		kvSchedulerEndpoint: kvSchedulerEndpoint,
	}, nil
}

func (c *kvSchedulerClient) downstreamResync() {
	downSteamResyncPath := c.kvSchedulerEndpoint + kvSchedulerPrefix + downStreamResync
	request, err := http.NewRequest("POST", downSteamResyncPath, nil)
	if err != nil {
		logrus.Errorf("kvSchedulerClient:, can't create request %v", err)
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		logrus.Errorf("kvSchedulerClient:, can't do request %v, error: %v", resp, err)
	}
	logrus.Infof("kvSchedulerClient: response %v from %v", resp, downSteamResyncPath)
}

func buildKvSchedulerDownStreamPath(vppAgentEndpoint string) (string, error) {
	parts := strings.Split(vppAgentEndpoint, ":")
	serverURL := fmt.Sprintf("http://%v:%v", parts[len(parts)-1], kvSchedulerPort)
	_, err := url.Parse(vppAgentEndpoint)
	if err != nil {
		return "", err
	}
	return serverURL, nil
}
