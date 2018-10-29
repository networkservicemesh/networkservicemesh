package nsmvpp

import (
	"fmt"
	"strconv"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

type vppInterface struct {
	mechanism *common.LocalMechanism // we want to save parameters here in order to recreate interface
	id        uint32
}

var (
	connections map[int][]operation = make(map[int][]operation)
	lastId      int                 = 0
)

// DreateLocalConnect sanity checks parameters passed in the LocalMechanisms and call nsmvpp.CreateLocalConnect
func CreateLocalConnect(apiCh govppapi.Channel, src, dst *common.LocalMechanism) (string, error) {
	srcIntf := &vppInterface{}
	dstIntf := &vppInterface{}

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

	pos, err := perform(tx, apiCh)
	if err != nil {
		rollback(tx, pos, apiCh)
		return "", err
	}

	lastId++
	connections[lastId] = tx // save transaction log to perform rollback on delete connection
	return fmt.Sprintf("%d", lastId), nil
}

// DeleteLocalConnect
func DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	id, _ := strconv.Atoi(connID)
	tx := connections[id]
	return rollback(tx, len(tx), apiCh)
}
