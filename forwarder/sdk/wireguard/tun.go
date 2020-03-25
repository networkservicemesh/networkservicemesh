// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package wireguard provides common API for wireguard devices.
package wireguard

import (
	"os"

	"golang.zx2c4.com/wireguard/device"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"golang.org/x/net/ipv4"

	"golang.zx2c4.com/wireguard/tun"
)

type tunL2Adapter struct {
	original tun.Device
}

func (c *tunL2Adapter) File() *os.File {
	return c.original.File()
}

func (c *tunL2Adapter) Read(buffer []byte, offset int) (int, error) {
	size, err := c.original.Read(buffer, offset)
	logrus.Infof("read data: %+v", buffer[:size])
	if err != nil {
		return size, err
	}
	return c.wrapL2Trafic(buffer, offset, size)
}

func (c *tunL2Adapter) wrapL2Trafic(buffer []byte, offset, size int) (int, error) {
	if size+ipv4.HeaderLen > len(buffer) {
		return size, errors.New("can not append transport header to buffer")
	}
	injectHeader := make([]byte, ipv4.HeaderLen)
	injectHeader[0] = ipv4.Version << 4
	injectHeader[device.IPv4offsetTotalLength+1] = ipv4.HeaderLen + byte(size)
	tmp := make([]byte, len(buffer)-offset-ipv4.HeaderLen)
	copy(tmp, buffer[offset:])
	copy(buffer[offset:], injectHeader)
	copy(buffer[offset+ipv4.HeaderLen:], tmp)
	return size + ipv4.HeaderLen, nil
}

func (c *tunL2Adapter) Write(buffer []byte, offset int) (int, error) {
	packet := buffer[offset:]
	transport := buffer[:offset]
	buffer = make([]byte, len(buffer)-ipv4.HeaderLen)
	copy(buffer, transport)
	copy(buffer[len(transport):], packet[ipv4.HeaderLen:])
	return c.original.Write(buffer, offset)
}

func (c *tunL2Adapter) Flush() error {
	return c.original.Flush()
}

func (c *tunL2Adapter) MTU() (int, error) {
	return c.original.MTU()
}

func (c *tunL2Adapter) Name() (string, error) {
	return c.original.Name()
}

func (c *tunL2Adapter) Events() chan tun.Event {
	return c.original.Events()
}

func (c *tunL2Adapter) Close() error {
	return c.original.Close()
}
