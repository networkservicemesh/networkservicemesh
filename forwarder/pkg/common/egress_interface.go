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
	SrcIPV6Net() *net.IPNet
	SrcLocalSID() net.IP
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
	srcV6Net          *net.IPNet
	localSID          net.IP
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
			return "", nil, errors.Wrap(err, "failed to read proc file")
		}
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, "\t")
		logrus.Printf("Parts: %v", parts)
		if len(parts) < 3 {
			return "", nil, errors.New("invalid line in proc file")
		}
		if strings.TrimSpace(parts[1]) == "00000000" {
			outgoingInterface := strings.TrimSpace(parts[0])
			defaultGateway := strings.TrimSpace(parts[2])
			ip, err := parseGatewayIP(defaultGateway)
			if err != nil {
				return "", nil, errors.WithMessagef(err, "error processing gateway IP %v for outgoing interface: %v", defaultGateway, outgoingInterface)

			}
			logrus.Printf("Found default gateway %v outgoing: %v", ip.String(), outgoingInterface)
			return outgoingInterface, ip, nil
		}
	}
	return "", nil, errors.New("failed to locate default route")
}

func parseGatewayIP(defaultGateway string) (net.IP, error) {
	if len(defaultGateway) != 8 {
		return nil, errors.New("failed to parse IP from string")
	}
	ip := net.IP{0, 0, 0, 0}
	for i := 0; i < len(defaultGateway)/2; i++ {
		iv, err := strconv.ParseInt(defaultGateway[i*2:i*2+2], 16, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "string does not represent a valid IP address")
		}
		ip[3-i] = byte(iv)
	}
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
func NewEgressInterface(srcIP net.IP) (EgressInterfaceType, net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	outgoingInterface, gw, err := findDefaultGateway4()
	if err != nil {
		return nil, nil, err
	}

	arpEntries, err := getArpEntries()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, err
		}
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		logrus.Infof("INTERFACE: %v : %v", iface.Name, addrs)
		if err != nil {
			return nil, nil, err
		}

		var v6 *net.IPNet
		// Some clouds require localSID to be equal local ipv6 address, or gateway will deny packets
		var localSID net.IP
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if strings.Contains(v.String(), ":") {
					if !(v.IP[0] == 0xfe && v.IP[1] == 0x80) {
						if v6 == nil {
							v6 = v
						} else {
							localSID = v.IP
						}
					}
				}
			}
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ipAddr := v.IP
				mask := v.Mask
				ipMask := ipAddr.Mask(mask)
				if v.IP.Equal(srcIP) || ipMask.Equal(srcIP) {
					if v6 != nil && localSID == nil {
						localSID = v6.IP
						v6.IP = make(net.IP, len(v6.IP))
						copy(v6.IP, localSID)
						v6.IP[0] = 0xfd
						v6.IP[1] = 0x24
					}

					return &egressInterface{
						srcNet:            v,
						srcV6Net:          v6,
						localSID:          localSID,
						iface:             &iface,
						defaultGateway:    gw,
						outgoingInterface: outgoingInterface,
						arpEntries:        arpEntries,
					}, v.IP, nil
				}
			default:
				return nil, nil, errors.New("type of addr not net.IPNET")
			}
		}
	}
	return nil, nil, errors.Errorf("unable to find interface with IP: %s", srcIP)
}

func (e *egressInterface) SrcIPNet() *net.IPNet {
	if e == nil {
		return nil
	}
	return e.srcNet
}

func (e *egressInterface) SrcIPV6Net() *net.IPNet {
	if e == nil {
		return nil
	}
	return e.srcV6Net
}

func (e *egressInterface) SrcLocalSID() net.IP {
	if e == nil {
		return nil
	}
	return e.localSID
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
