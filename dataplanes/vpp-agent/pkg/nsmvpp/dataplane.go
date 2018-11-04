package nsmvpp

type VPPAgentClient struct {
}

func NewVPPAgentClient() *VPPAgentClient {
	return &VPPAgentClient{}
}

func (client *VPPAgentClient) IsConnected() bool {
	return true
}

func (client *VPPAgentClient) Shutdown() {

}
