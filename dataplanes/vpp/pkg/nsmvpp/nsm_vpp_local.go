package nsmvpp

import (
	"fmt"
	"strconv"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

type vppInterface struct {
	mechanism *common.LocalMechanism // we want to save parameters here in order to recreate interface
	socketId  uint32
	id        uint32
}

var (
	connections map[uint32][]operation = make(map[uint32][]operation)
	lastId      uint32                 = 1
)

// CreateLocalConnect sanity checks parameters passed in the LocalMechanisms and call nsmvpp.CreateLocalConnect
func CreateLocalConnect(apiCh govppapi.Channel, src, dst *common.LocalMechanism) (string, error) {
	connectionId := lastId
	srcIntf := &vppInterface{socketId: connectionId}
	dstIntf := &vppInterface{socketId: connectionId + 1}
	lastId += 2 // each connection use two ids for sockets

	tx := []operation{
		&createLocalInterface{
			localMechanism: src,
			intf:           srcIntf,
		},
		&createLocalInterface{
			localMechanism: dst,
			intf:           dstIntf,
		},
		&interfaceXconnect{
			rx:     srcIntf,
			tx:     dstIntf,
			enable: 1,
		},
		&interfaceXconnect{
			rx:     dstIntf,
			tx:     srcIntf,
			enable: 1,
		},
		&interfaceUpDown{
			intf:   srcIntf,
			upDown: 1,
		},
		&interfaceUpDown{
			intf:   dstIntf,
			upDown: 1,
		},
	}

	if (src.Type == common.LocalMechanismType_MEM_INTERFACE) && (dst.Type == common.LocalMechanismType_MEM_INTERFACE) {
		var err error
		if tx, err = memifDirectConnect(src.Parameters, dst.Parameters); err != nil {
			return "", err
		}
	}

	pos, err := perform(tx, apiCh)
	if err != nil {
		rollback(tx, pos, apiCh)
		return "", err
	}

	connections[connectionId] = tx // save transaction log to perform rollback on delete connection
	return fmt.Sprintf("%d", connectionId), nil
}

// DeleteLocalConnect
func DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	id64, _ := strconv.ParseUint(connID, 10, 32)
	id := uint32(id64)
	tx := connections[id]
	return rollback(tx, len(tx), apiCh)
}
