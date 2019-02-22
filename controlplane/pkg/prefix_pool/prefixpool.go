package prefix_pool

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"math/big"
	"net"
	"sync"
)

/**
All in one interface for managing prefixes.
*/
type PrefixPool interface {
	/*
		Process ExtraPrefixesRequest and provide a list of prefixes for clients to use.
	*/
	Extract(connectionId string, family connectioncontext.IpFamily_Family, requests ...*connectioncontext.ExtraPrefixRequest) (srcIP *net.IPNet, dstIP *net.IPNet, requested []string, err error)
	Release(connectionId string) error
	GetConnectionInformation(connectionId string) (string, []string, error)
	GetPrefixes() []string
}
type prefixPool struct {
	sync.RWMutex

	basePrefixes []string // Just to know where we start from
	prefixes     []string
	connections  map[string]*connectionRecord
}

func (impl *prefixPool) GetPrefixes() []string {
	return impl.prefixes
}

type connectionRecord struct {
	ipNet    *net.IPNet
	prefixes []string
}

func NewPrefixPool(prefixes ...string) (PrefixPool, error) {
	//TODO: Add validation of input prefixes.
	return &prefixPool{
		basePrefixes: prefixes,
		prefixes:     prefixes,
		connections:  map[string]*connectionRecord{},
	}, nil
}

func (impl *prefixPool) Extract(connectionId string, family connectioncontext.IpFamily_Family, requests ...*connectioncontext.ExtraPrefixRequest) (srcIP *net.IPNet, dstIP *net.IPNet, requested []string, err error) {
	impl.Lock()
	defer impl.Unlock()

	prefixLen := 30 // At lest 4 addresses
	if family == connectioncontext.IpFamily_IPV6 {
		prefixLen = 126
	}
	result, remaining, err := ExtractPrefixes(impl.prefixes, &connectioncontext.ExtraPrefixRequest{
		RequiredNumber:  1,
		RequestedNumber: 1,
		PrefixLen:       uint32(prefixLen),
		AddrFamily:      &connectioncontext.IpFamily{Family: family},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	ip, ipNet, err := net.ParseCIDR(result[0])
	if err != nil {
		return nil, nil, nil, err
	}

	src, err := IncrementIP(ip, ipNet)
	if err != nil {
		return nil, nil, nil, err
	}

	dst, err := IncrementIP(src, ipNet)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(requests) > 0 {
		requested, remaining, err = ExtractPrefixes(remaining, requests...)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	impl.prefixes = remaining

	impl.connections[connectionId] = &connectionRecord{
		ipNet:    ipNet,
		prefixes: requested,
	}
	return &net.IPNet{IP: src, Mask: ipNet.Mask}, &net.IPNet{IP: dst, Mask: ipNet.Mask}, requested, nil
}
func (impl *prefixPool) Release(connectionId string) error {
	impl.Lock()
	defer impl.Unlock()

	conn := impl.connections[connectionId]
	if conn == nil {
		return fmt.Errorf("Failed to release connection infomration: %s", connectionId)
	}
	delete(impl.connections, connectionId)

	remaining, err := ReleasePrefixes(impl.prefixes, conn.prefixes...)
	if err != nil {
		return err
	}

	remaining, err = ReleasePrefixes(remaining, conn.ipNet.String())
	if err != nil {
		return err
	}

	impl.prefixes = remaining
	return nil
}

func (impl *prefixPool) GetConnectionInformation(connectionId string) (string, []string, error) {
	impl.RLock()
	defer impl.RUnlock()
	conn := impl.connections[connectionId]
	if conn == nil {
		return "", nil, fmt.Errorf("No connection with id: %s is found", connectionId)
	}
	return conn.ipNet.String(), conn.prefixes, nil
}

func ExtractPrefixes(prefixes []string, requests ...*connectioncontext.ExtraPrefixRequest) (requested []string, remaining []string, err error) {
	// Check if requests are valid.

	for _, request := range requests {
		error := request.IsValid()
		if error != nil {
			return nil, prefixes, error
		}
	}

	//1 find prefix of desired prefix len and return it
	result := []string{}

	// Make a copy since we need to fit all prefixes before we could finish.
	newPrefixes := []string{}
	newPrefixes = append(newPrefixes, prefixes...)

	// We need to firstly find required prefixes available.
	for _, request := range requests {
		for i := uint32(0); i < request.RequiredNumber; i++ {
			prefix, leftPrefixes, error := ExtractPrefix(newPrefixes, request.PrefixLen)
			if error != nil {
				return nil, prefixes, error
			}
			result = append(result, prefix)
			newPrefixes = leftPrefixes
		}
	}
	// We need to fit some more prefies up to Requested ones
	for _, request := range requests {
		for i := request.RequiredNumber; i < request.RequestedNumber; i++ {
			prefix, leftPrefixes, error := ExtractPrefix(newPrefixes, request.PrefixLen)
			if error != nil {
				// It seems there is no more prefixes available, but since we have all Required already we could go.
				break
			}
			result = append(result, prefix)
			newPrefixes = leftPrefixes
		}
	}
	if len(result) == 0 {
		return nil, prefixes, fmt.Errorf("Failed to extract prefixes, there is no available %v", prefixes)
	}
	return result, newPrefixes, nil
}

func ExtractPrefix(prefixes []string, prefixLen uint32) (string, []string, error) {
	// Check if we already have required CIDR
	max_prefix := 0
	max_prefix_idx := -1

	// Check if we already have required prefix,
	for idx, prefix := range prefixes {
		_, netip, error := net.ParseCIDR(prefix)
		if error != nil {
			continue
		}
		parentLen, _ := netip.Mask.Size()
		// Check if some of requests are fit into this prefix.
		if prefixLen == uint32(parentLen) {
			// We found required one.
			resultPrefix := prefixes[idx]
			resultPrefixes := append(prefixes[:idx], prefixes[idx+1:]...)
			// Lets remove from list and return
			return resultPrefix, resultPrefixes, nil
		} else {
			// Update minimal root for split
			if uint32(parentLen) < prefixLen && (parentLen > max_prefix || max_prefix == 0) {
				max_prefix = parentLen
				max_prefix_idx = idx
			}
		}
	}
	// Not found, lets split minimal found prefix
	if max_prefix_idx == -1 {
		// There is no room to split
		return "", prefixes, fmt.Errorf("Failed to find room to have prefix len %d at %v", prefixLen, prefixes)
	}

	resultPrefixRoot := prefixes[max_prefix_idx]
	right_parts := []string{}

	_, rootCIDRNet, _ := net.ParseCIDR(resultPrefixRoot)
	for {
		rootLen, _ := rootCIDRNet.Mask.Size()
		if uint32(rootLen) == prefixLen {
			// we found required prefix
			break
		}
		sub1, error := subnet(rootCIDRNet, 0)
		if error != nil {
			return "", prefixes, error
		}

		sub2, error := subnet(rootCIDRNet, 1)
		if error != nil {
			return "", prefixes, error
		}
		right_parts = append(right_parts, sub2.String())
		rootCIDRNet = sub1
	}
	var resultPrefixes []string = nil
	resultPrefixes = append(resultPrefixes, prefixes[:max_prefix_idx]...)
	resultPrefixes = append(resultPrefixes, reverse(right_parts)...)
	resultPrefixes = append(resultPrefixes, prefixes[max_prefix_idx+1:]...)

	// return result
	return rootCIDRNet.String(), resultPrefixes, nil
}

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] != true {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}
	return result
}

func reverse(values []string) []string {
	newValues := make([]string, len(values))

	for i, j := 0, len(values)-1; i <= j; i, j = i+1, j-1 {
		newValues[i] = values[j]
		if i != j {
			newValues[j] = values[i]
		}
	}
	return newValues
}

func ReleasePrefixes(prefixes []string, released ...string) (remaining []string, err error) {
	result := []string{}
	if released == nil || len(released) == 0 {
		return prefixes, nil
	}
	result = removeDuplicates(append(prefixes, released...))

	prefixByPrefixLen := map[int][]*net.IPNet{}

	for _, prefix := range result {
		_, ipnet, error := net.ParseCIDR(prefix)
		if error != nil {
			return nil, error
		}
		parentLen, _ := ipnet.Mask.Size()
		nets := prefixByPrefixLen[parentLen]
		nets = append(nets, ipnet)
		prefixByPrefixLen[parentLen] = nets
	}

	for {
		newPrefixByPrefixLen := map[int][]*net.IPNet{}
		changes := 0
		for basePrefix, value := range prefixByPrefixLen {
			base := map[string]*net.IPNet{}

			cvalue := append(newPrefixByPrefixLen[basePrefix], value...)
			parentPrefix := []*net.IPNet{}
			if len(cvalue) < 2 {
				newPrefixByPrefixLen[basePrefix] = cvalue
				continue
			}
			for _, netValue := range cvalue {
				baseIp := &net.IPNet{
					IP:   clearNetIndexInIP(netValue.IP, basePrefix),
					Mask: netValue.Mask,
				}
				parentLen, addrLen := baseIp.Mask.Size()
				bv := base[baseIp.String()]
				if bv == nil {
					base[baseIp.String()] = netValue
				} else {
					// We found item with same base IP, we we could join this two networks into one.
					// Remove from current level
					delete(base, baseIp.String())
					// And put to next level
					parentPrefix = append(parentPrefix, &net.IPNet{
						IP:   baseIp.IP,
						Mask: net.CIDRMask(parentLen-1, addrLen),
					})
					changes++
				}
			}
			leftPrefixes := []*net.IPNet{}
			// Put all not merged values
			for _, value := range base {
				leftPrefixes = append(leftPrefixes, value)
			}
			newPrefixByPrefixLen[basePrefix] = leftPrefixes
			newPrefixByPrefixLen[basePrefix-1] = append(newPrefixByPrefixLen[basePrefix-1], parentPrefix...)
		}
		if changes == 0 {
			// All is merged, we could exit
			result = []string{}
			for _, value := range newPrefixByPrefixLen {
				for _, v := range value {
					result = append(result, v.String())
				}
			}
			return result, nil
		}
		prefixByPrefixLen = newPrefixByPrefixLen
	}
}

func subnet(ipnet *net.IPNet, subnet_index int) (*net.IPNet, error) {
	mask := ipnet.Mask

	parentLen, addrLen := mask.Size()
	newPrefixLen := parentLen + 1
	if newPrefixLen > addrLen {
		return nil, fmt.Errorf("insufficient address space to extend prefix of %d", parentLen)
	}

	if uint64(subnet_index) > 2 {
		return nil, fmt.Errorf("prefix extension does not accommodate a subnet numbered %d", subnet_index)
	}

	return &net.IPNet{
		IP:   setNetIndexInIP(ipnet.IP, subnet_index, newPrefixLen),
		Mask: net.CIDRMask(newPrefixLen, addrLen),
	}, nil
}

func AddressCount(prefixes ...string) uint64 {
	var c uint64 = 0
	for _, pr := range prefixes {
		c += addressCount(pr)
	}
	return c
}

func addressCount(pr string) uint64 {
	_, network, _ := net.ParseCIDR(pr)
	prefixLen, bits := network.Mask.Size()
	return 1 << (uint64(bits) - uint64(prefixLen))
}

func setNetIndexInIP(ip net.IP, num int, prefixLen int) net.IP {
	ipInt, totalBits := fromIP(ip)
	bigNum := big.NewInt(int64(num))
	bigNum.Lsh(bigNum, uint(totalBits-prefixLen))
	ipInt.Or(ipInt, bigNum)
	return toIP(ipInt, totalBits)
}

func clearNetIndexInIP(ip net.IP, prefixLen int) net.IP {
	ipInt, totalBits := fromIP(ip)
	ipInt.SetBit(ipInt, totalBits-prefixLen, 0)
	return toIP(ipInt, totalBits)
}

func toIP(ipInt *big.Int, bits int) net.IP {
	ipBytes := ipInt.Bytes()
	ret := make([]byte, bits/8)
	// Pack our IP bytes into the end of the return array,
	// since big.Int.Bytes() removes front zero padding.
	for i := 1; i <= len(ipBytes); i++ {
		ret[len(ret)-i] = ipBytes[len(ipBytes)-i]
	}
	return net.IP(ret)
}
func fromIP(ip net.IP) (*big.Int, int) {
	val := &big.Int{}
	val.SetBytes([]byte(ip))
	i := len(ip)
	if i == net.IPv4len {
		return val, 32
	} // else if i == net.IPv6len
	return val, 128
}

/**
AddressRange returns the first and last addresses in the given CIDR range.
*/
func AddressRange(network *net.IPNet) (net.IP, net.IP) {
	firstIP := network.IP
	// the last IP is the network address OR NOT the mask address
	prefixLen, bits := network.Mask.Size()
	if prefixLen == bits {
		lastIP := make([]byte, len(firstIP))
		copy(lastIP, firstIP)
		return firstIP, lastIP
	}

	firstIPInt, bits := fromIP(firstIP)
	hostLen := uint(bits) - uint(prefixLen)
	lastIPInt := big.NewInt(1)
	lastIPInt.Lsh(lastIPInt, hostLen)
	lastIPInt.Sub(lastIPInt, big.NewInt(1))
	lastIPInt.Or(lastIPInt, firstIPInt)

	return firstIP, toIP(lastIPInt, bits)
}

func IncrementIP(sourceIp net.IP, ipNet *net.IPNet) (net.IP, error) {
	ip := make([]byte, len(sourceIp))
	copy(ip, sourceIp)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	if !ipNet.Contains(ip) {
		return nil, fmt.Errorf("Overflowed CIDR while incrementing IP")
	}
	return ip, nil
}
