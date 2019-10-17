// Copyright (c) 2019 Cisco and/or its affiliates.
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

package common

import (
	"bufio"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

type ARPEntry struct {
	Interface   string
	IPAddress   string
	PhysAddress string
}

// EgressInterfaceType describes the info about the egress interface used for tunneling
type EgressInterfaceType interface {
	SrcIPNet() *net.IPNet
	DefaultGateway() *net.IP
	Interface() *net.Interface
	Name() string
	HardwareAddr() *net.HardwareAddr
	OutgoingInterface() string
	ArpEntries() []*ARPEntry
}

type egressInterface struct {
	EgressInterfaceType
	srcNet            *net.IPNet
	iface             *net.Interface
	defaultGateway    net.IP
	outgoingInterface string
	arpEntries        []*ARPEntry
}

func findDefaultGateway4() (string, net.IP, error) {
	f, err := os.OpenFile("/proc/net/route", os.O_RDONLY, 0600)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	return parseProcFile(scanner)
}

func parseProcFile(scanner *bufio.Scanner) (string, net.IP, error) {
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			logrus.Errorf("Failed to read routes files: %v", err)
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, "\t")
		if strings.TrimSpace(parts[1]) == "00000000" {
			outgoingInterface := strings.TrimSpace(parts[0])
			defaultGateway := strings.TrimSpace(parts[2])
			ip, err := parseGatewayIP(defaultGateway)
			if err != nil {
				logrus.Printf("Error Processing gateway IP %v for outgoing interface: %v", defaultGateway, outgoingInterface)
				break
			}
			logrus.Printf("Found default gateway %v outgoing: %v", ip.String(), outgoingInterface)
			return outgoingInterface, ip, nil
		}
	}
	return "", nil, errors.New("Failed to locate default route...")
}

func parseGatewayIP(defaultGateway string) (net.IP, error) {
	ip := net.IP{0, 0, 0, 0}
	if len(defaultGateway) != 8 {
		return ip, errors.New("Failed to parse IP from string")
	}
	iv0, _ := strconv.ParseInt(defaultGateway[0:2], 16, 32)
	iv1, _ := strconv.ParseInt(defaultGateway[2:4], 16, 32)
	iv2, _ := strconv.ParseInt(defaultGateway[4:6], 16, 32)
	iv3, _ := strconv.ParseInt(defaultGateway[6:], 16, 32)
	ip[0] = byte(iv3)
	ip[1] = byte(iv2)
	ip[2] = byte(iv1)
	ip[3] = byte(iv0)
	return ip, nil

}

func getArpEntries() ([]*ARPEntry, error) {
	f, err := os.OpenFile("/proc/net/arp", os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	arps := []*ARPEntry{}
	for l := 0; ; l++ {
		line, err := reader.ReadString('\n')

		if err != nil {
			if err != io.EOF {
				break
			}
			break
		}

		if l == 0 {
			continue //Skip first line with headers and empty line
		}
		if line == "" {
			break //Skip first line with headers and empty line
		}
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		arps = append(arps, &ARPEntry{
			PhysAddress: strings.TrimSpace(parts[3]),
			IPAddress:   strings.TrimSpace(parts[0]),
			Interface:   strings.TrimSpace(parts[5]),
		})
	}
	return arps, nil
}

// NewEgressInterface creates a new egress interface object
func NewEgressInterface(srcIP net.IP) (EgressInterfaceType, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	outgoingInterface, gw, err := findDefaultGateway4()
	if err != nil {
		return nil, err
	}

	arpEntries, err := getArpEntries()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.Equal(srcIP) {
					return &egressInterface{
						srcNet:            v,
						iface:             &iface,
						defaultGateway:    gw,
						outgoingInterface: outgoingInterface,
						arpEntries:        arpEntries,
					}, nil
				}
			default:
				return nil, errors.New("Type of addr not net.IPNET")
			}
		}
	}
	return nil, errors.Errorf("unable to find interface with IP: %s", srcIP)
}

func (e *egressInterface) SrcIPNet() *net.IPNet {
	if e == nil {
		return nil
	}
	return e.srcNet
}

func (e *egressInterface) Interface() *net.Interface {
	if e == nil {
		return nil
	}
	return e.iface
}

func (e *egressInterface) DefaultGateway() *net.IP {
	if e == nil {
		return nil
	}
	return &e.defaultGateway
}

func (e *egressInterface) Name() string {
	if e == nil {
		return ""
	}
	return e.Interface().Name
}

func (e *egressInterface) HardwareAddr() *net.HardwareAddr {
	if e == nil {
		return nil
	}
	return &e.Interface().HardwareAddr
}

func (e *egressInterface) OutgoingInterface() string {
	if e == nil {
		return ""
	}
	return e.outgoingInterface
}

func (e *egressInterface) ArpEntries() []*ARPEntry {
	if e == nil {
		return nil
	}
	return e.arpEntries
}
