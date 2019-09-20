package prefix_pool

import (
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
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
	Intersect(prefix string) (bool, error)
	ExcludePrefixes(excludedPrefixes []string) ([]string, error)
	ReleaseExcludedPrefixes(excludedPrefixes []string) error
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

/* Release excluded prefixes back the pool of available ones */
func (impl *prefixPool) ReleaseExcludedPrefixes(excludedPrefixes []string) error {
	impl.Lock()
	defer impl.Unlock()

	remaining, err := ReleasePrefixes(impl.prefixes, excludedPrefixes...)
	if err != nil {
		return err
	}
	/* Sort the prefixes, so their order is consistent during unit testing */
	sort.Slice(remaining, func(i, j int) bool { return remaining[i] < remaining[j] })
	impl.prefixes = remaining
	return nil
}

/* Exclude prefixes from the pool of available prefixes */
func (impl *prefixPool) ExcludePrefixes(excludedPrefixes []string) ([]string, error) {
	impl.Lock()
	defer impl.Unlock()
	/* Use a working copy for the available prefixes */
	copyPrefixes := append([]string{}, impl.prefixes...)

	removedPrefixes := []string{}

	for _, excludedPrefix := range excludedPrefixes {
		splittedEntries := []string{}
		prefixesToRemove := []string{}
		_, subnetExclude, err := net.ParseCIDR(excludedPrefix)
		if err != nil {
			err := fmt.Errorf("IPAM+: Unable to parse excluded prefix: %s", excludedPrefix)
			logrus.Errorf("%v", err)
			return nil, err
		}

		/* 1. Check if each excluded entry overlaps with the available prefix */
		for _, prefix := range copyPrefixes {
			intersecting := false
			excludedIsBigger := false
			_, subnetPrefix, err := net.ParseCIDR(prefix)
			if err != nil {
				err := fmt.Errorf("IPAM-: Unable to parse available prefix: %s", prefix)
				logrus.Errorf("%v", err)
				return nil, err
			}
			intersecting, excludedIsBigger = intersect(subnetExclude, subnetPrefix)
			/* 1.1. If intersecting, check which one is bigger */
			if intersecting {
				/* 1.1.1. If excluded is bigger, we remove the original entry */
				if !excludedIsBigger {
					/* 1.1.2. If the original entry is bigger, we split it and remove the avoided range */
					res, err := extractSubnet(subnetPrefix, subnetExclude)
					if err != nil {
						return nil, err
					}
					/* 1.1.3. Collect the resulted split prefixes */
					splittedEntries = append(splittedEntries, res...)
					/* 1.1.5. Collect the actual excluded prefixes that should be added back to the original pool */
					removedPrefixes = append(removedPrefixes, subnetExclude.String())
				} else {
					/* 1.1.5. Collect the actual excluded prefixes that should be added back to the original pool */
					removedPrefixes = append(removedPrefixes, subnetPrefix.String())
				}
				/* 1.1.4. Collect prefixes that should be removed from the original pool */
				prefixesToRemove = append(prefixesToRemove, subnetPrefix.String())

				break
			}
			/* 1.2. If not intersecting, proceed verifying the next one */
		}
		/* 2. Keep only the prefixes that should not be removed from the original pool */
		if len(prefixesToRemove) != 0 {
			for _, presentPrefix := range copyPrefixes {
				prefixFound := false
				for _, prefixToRemove := range prefixesToRemove {
					if presentPrefix == prefixToRemove {
						prefixFound = true
						break
					}
				}
				if !prefixFound {
					/* 2.1. Add the original non-split prefixes to the split ones */
					splittedEntries = append(splittedEntries, presentPrefix)
				}
			}
			/* 2.2. Update the original prefix list */
			copyPrefixes = splittedEntries
		}
	}
	/* Raise an error, if there aren't any available prefixes left after excluding */
	if len(copyPrefixes) == 0 {
		err := fmt.Errorf("IPAM: The available address pool is empty, probably intersected by excludedPrefix")
		logrus.Errorf("%v", err)
		return nil, err
	}
	/* Everything should be fine, update the available prefixes with what's left */
	impl.prefixes = copyPrefixes
	return removedPrefixes, nil
}

/* Split the wider range removing the avoided smaller range from it */
func extractSubnet(wider, smaller *net.IPNet) ([]string, error) {
	root := wider
	prefixLen, _ := smaller.Mask.Size()
	leftParts, rightParts := []string{}, []string{}
	for {
		rootLen, _ := root.Mask.Size()
		if rootLen == prefixLen {
			// we found the required prefix
			break
		}
		sub1, err := subnet(root, 0)
		if err != nil {
			return nil, err
		}
		sub2, err := subnet(root, 1)
		if err != nil {
			return nil, err
		}
		if sub1.Contains(smaller.IP) {
			rightParts = append(rightParts, sub2.String())
			root = sub1
		} else if sub2.Contains(smaller.IP) {
			leftParts = append(leftParts, sub1.String())
			root = sub2
		} else {
			return nil, fmt.Errorf("split failed")
		}
	}
	return append(leftParts, rightParts...), nil
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

func (impl *prefixPool) Intersect(prefix string) (bool, error) {
	_, subnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return false, err
	}

	for _, p := range impl.prefixes {
		_, sn, _ := net.ParseCIDR(p)
		if ret, _ := intersect(sn, subnet); ret {
			return true, nil
		}
	}
	return false, nil
}

func intersect(first, second *net.IPNet) (bool, bool) {
	f, _ := first.Mask.Size()
	s, _ := second.Mask.Size()
	firstIsBigger := false

	var widerRange, narrowerRange *net.IPNet
	if f < s {
		widerRange, narrowerRange = first, second
		firstIsBigger = true
	} else {
		widerRange, narrowerRange = second, first
	}

	return widerRange.Contains(narrowerRange.IP), firstIsBigger
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
		if !encountered[elements[v]] {
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
	if len(released) == 0 {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxCommonPrefixSubnet(s1, s2 *net.IPNet) *net.IPNet {
	rawIp1, n1 := fromIP(s1.IP)
	rawIp2, _ := fromIP(s2.IP)

	xored := &big.Int{}
	xored.Xor(rawIp1, rawIp2)
	maskSize := leadingZeros(xored, n1)

	m1, bits := s1.Mask.Size()
	m2, _ := s2.Mask.Size()

	mask := net.CIDRMask(min(min(m1, m2), maskSize), bits)
	return &net.IPNet{
		IP:   s1.IP.Mask(mask),
		Mask: mask,
	}
}

func IpToNet(ipAddr net.IP) *net.IPNet {
	mask := net.CIDRMask(len(ipAddr)*8, len(ipAddr)*8)
	return &net.IPNet{IP: ipAddr, Mask: mask}
}

func leadingZeros(n *big.Int, size int) int {
	i := size - 1
	for ; n.Bit(i) == 0 && i > 0; i-- {
	}
	return size - 1 - i
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
