package sid

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

const SIDPrefix = "fd25::"

type SIDAllocator interface {
	SID(requestID string) string
	Restore(hardwareAddr, sid string)
}

type sidAllocator struct {
	lastSID map[string]uint32
	sync.Mutex
}

func NewSIDAllocator() SIDAllocator {
	return &sidAllocator{
		lastSID: make(map[string]uint32),
	}
}

// SID - Allocate a new SID for SRv6 Policy
func (a *sidAllocator) SID(requestID string) string {
	lastSID := a.lastSID[requestID] + 1
	if lastSID < 2 {
		lastSID = 2
	}

	a.lastSID[requestID] = lastSID
	return fmt.Sprintf("%s:%x", transformRequestID(requestID), lastSID)
}

// Restore value of last SID based on connections we have at the moment
func (a *sidAllocator) Restore(requestID, sid string) {
	parsedSID := net.ParseIP(sid)
	intSID := binary.BigEndian.Uint16(parsedSID[len(parsedSID)-2:])
	a.lastSID[requestID] = uint32(intSID)
}

func GenerateHostIP(requestID string) string {
	return fmt.Sprintf("%s%x", transformRequestID(requestID), 1)
}

func transformRequestID(requestID string) string {
	sid := requestID
	for i := 0; i < len(requestID)/4; i++ {
		idx := len(requestID) - (i+1)*4
		sid = fmt.Sprintf("%s:%s", sid[:idx], sid[idx:])
	}

	logrus.Printf("Generated IP: %v %v", fmt.Sprintf("%s%s", SIDPrefix, sid), net.ParseIP(fmt.Sprintf("%s%s", SIDPrefix, sid)).String())

	return net.ParseIP(fmt.Sprintf("%s%s", SIDPrefix, sid)).String()
}
