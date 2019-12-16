// Copyright (c) 2019 VMware, Inc.
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

package metrics

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
)

// GetMetricsIdentifiers returns source and destination
// of the metrics specified with `pod` name and `namespace`
func GetMetricsIdentifiers(crossConnect *crossconnect.CrossConnect) (map[string]string, error) {
	srcPod, srcNamespace, dstPod, dstNamespace := "", "", "", ""
	if crossConnect.GetSource() == nil {
		return nil, errors.Errorf("error: crossConnect should have at least one source, %v", crossConnect)
	}

	ccSrcLabels := crossConnect.GetSource().GetLabels()
	srcPod = ccSrcLabels[connection.PodNameKey]
	srcNamespace = ccSrcLabels[connection.NamespaceKey]

	if crossConnect.GetDestination() == nil {
		return nil, errors.Errorf("error: crossConnect should have at least one destination, %v", crossConnect)
	}

	ccDstLabels := crossConnect.GetDestination().GetLabels()
	dstPod = ccDstLabels[connection.PodNameKey]
	dstNamespace = ccDstLabels[connection.NamespaceKey]

	res := map[string]string{
		SrcPodKey:       srcPod,
		SrcNamespaceKey: srcNamespace,
		DstPodKey:       dstPod,
		DstNamespaceKey: dstNamespace,
	}

	return res, nil
}
