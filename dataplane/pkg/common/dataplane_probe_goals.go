package common

import "fmt"

const (
	newEgressIPReady = 1 << iota
	srcIPReady
	socketCleanReady
	validIPReady
	socketListenReady
	done = newEgressIPReady | srcIPReady | socketCleanReady | validIPReady | socketListenReady
)

//DataplaneProbeGoals represents probes goals for Dataplane
type DataplaneProbeGoals struct {
	state int8
}

//Status returns current goals status
func (g *DataplaneProbeGoals) Status() string {
	return fmt.Sprintf("NewEgressIPReady:%v, SetSrcIPReady: %v, SetSocketCleanReady: %v, SetValidIPReady: %v, SetSocketListenrReady: %v",
		g.state&newEgressIPReady > 0,
		g.state&srcIPReady > 0,
		g.state&socketCleanReady > 0,
		g.state&validIPReady > 0,
		g.state&socketListenReady > 0)
}

//SetNewEgressIFReady sets true for NewEgressIFReady
func (g *DataplaneProbeGoals) SetNewEgressIFReady() {
	g.state |= newEgressIPReady
}

//IsComplete if all goals have done
func (g *DataplaneProbeGoals) IsComplete() bool {
	return g.state == done
}

//SetSrcIPReady sets true for SrcIPReady
func (g *DataplaneProbeGoals) SetSrcIPReady() {
	g.state |= srcIPReady
}

//SetSocketCleanReady sets true for SocketCleanReady
func (g *DataplaneProbeGoals) SetSocketCleanReady() {
	g.state |= socketCleanReady
}

//SetValidIPReady sets true for ValidIPReady
func (g *DataplaneProbeGoals) SetValidIPReady() {
	g.state |= validIPReady
}

//SetSocketListenReady sets true for SocketListenReady
func (g *DataplaneProbeGoals) SetSocketListenReady() {
	g.state |= socketListenReady
}
