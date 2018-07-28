// Copyright (c) 2018 Cisco and/or its affiliates.
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

package utils

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// nsmFinalizerName defines a finalizer string which gets added or removed to
	// NSM pod with injected dataplane interfaces
	nsmFinalizerName = "finalizer.networkservicemesh.io/nsm"
)

func findElement(ss []string, e string) (int, bool) {
	for i, s := range ss {
		if s == e {
			return i, true
		}
	}
	return 0, false
}

// AddPodFinalizer adds a specified finalizer to a pod
func AddPodFinalizer(k8s kubernetes.Interface, pn, pns string) error {
	tp, err := k8s.CoreV1().Pods(pns).Get(pn, meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	newFinalizers := tp.GetFinalizers()
	fn := nsmFinalizerName
	_, found := findElement(newFinalizers, fn)
	if !found {
		newFinalizers = append(newFinalizers, fn)
	} else {
		return nil
	}
	tp.SetFinalizers(newFinalizers)
	_, err = k8s.CoreV1().Pods(pns).Update(tp)

	return err
}

// RemovePodFinalizer removes a specified finalizer from a pod
func RemovePodFinalizer(k8s kubernetes.Interface, pn, pns string) error {
	tp, err := k8s.CoreV1().Pods(pns).Get(pn, meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	newFinalizers := tp.GetFinalizers()
	fn := nsmFinalizerName
	i, found := findElement(newFinalizers, fn)
	if found {
		newFinalizers = append(newFinalizers[:i], newFinalizers[i+1:]...)
	} else {
		return nil
	}
	tp.SetFinalizers(newFinalizers)
	_, err = k8s.CoreV1().Pods(pns).Update(tp)

	return err
}
