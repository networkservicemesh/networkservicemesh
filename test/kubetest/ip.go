// Copyright 2019 VMware, Inc.
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

package kubetest

import (
	"net"
	"strings"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func splitIP(ipResponse string, ipv6 bool) [2]string {
	var inet, srcIPStr string
	if !ipv6 {
		inet = "inet"
	} else {
		inet = "inet6"
	}
	a := strings.SplitAfterN(ipResponse, inet+" ", 2)
	b := strings.SplitAfterN(a[len(a)-1], "/", 2)
	if len(b) == 1 {
		srcIPStr = b[0]
	} else {
		srcIPStr = b[0][0 : len(b[0])-1]
	}
	srcIP := net.ParseIP(srcIPStr)
	dstIP := srcIP

	if !ipv6 {
		srcIP = srcIP.To4()
		dstIP = srcIP
		dstIP[3]++
	} else {
		srcIP = srcIP.To16()
		dstIP = srcIP
		dstIP[15]++
	}
	dstIPStr := dstIP.String()
	return [2]string{srcIPStr, dstIPStr}
}

func getNSCLocalRemoteIPs(k8s *K8s, nscPodNode *v1.Pod) [2]string {
	var ipResponse, errOut string
	var err error
	if !k8s.UseIPv6() {
		ipResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr", "show", "nsm0", "scope", "global")
	} else {
		ipResponse, errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "addr", "show", "nsm0", "scope", "global")
	}
	k8s.g.Expect(err).To(BeNil())
	k8s.g.Expect(errOut).To(Equal(""))
	result := splitIP(ipResponse, k8s.UseIPv6())
	return result
}
