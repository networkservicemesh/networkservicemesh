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

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmvpp"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	dataplaneapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	dataplane              = "/var/lib/networkservicemesh/nsm-vpp.dataplane.sock"
	interfaceNameMaxLength = 15
)

var (
	srcPodName      = flag.String("src-pod-name", "", "Name of the source pod")
	srcPodNamespace = flag.String("src-pod-namespace", "default", "Namespace of the source pod")
	dstPodName      = flag.String("dst-pod-name", "", "Name of the destination pod")
	dstPodNamespace = flag.String("dst-pod-namespace", "default", "Namespace of the destination pod")
	kubeconfig      = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

func buildClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		kubeconfig = &kubeconfigEnv
	}

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8s, nil
}

func getContainerID(k8s *kubernetes.Clientset, pn, ns string) (string, error) {
	pl, err := k8s.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, p := range pl.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, pn) {
			// Two cases main container is in Running state and in Pending
			// Pending can inidcate that Init container is still running
			if p.Status.Phase == v1.PodRunning {
				return strings.Split(p.Status.ContainerStatuses[0].ContainerID, "://")[1][:12], nil
			}
			if p.Status.Phase == v1.PodPending {
				// Check if we have Init containers
				if p.Status.InitContainerStatuses != nil {
					for _, i := range p.Status.InitContainerStatuses {
						if i.State.Running != nil {
							return strings.Split(i.ContainerID, "://")[1][:12], nil
						}
					}
				}
			}
			return "", fmt.Errorf("none of containers of pod %s/%s is in running state", p.ObjectMeta.Namespace, p.ObjectMeta.Name)
		}
	}

	return "", fmt.Errorf("pod %s/%s not found", ns, pn)
}

func init() {
	runtime.LockOSThread()
}

type dataplaneClientTest struct {
	testName         string
	localSource      *common.LocalMechanism
	localDestination *dataplaneapi.Connection_Local
	shouldFail       bool
}

func main() {
	flag.Parse()

	if *srcPodName == "" || *dstPodName == "" {
		logrus.Fatal("Both source and destination PODs' name must be specified, exitting...")
	}
	k8s, err := buildClient()
	if err != nil {
		logrus.Fatal("Failed to build kubernetes client, exitting...")
	}
	if _, err := os.Stat(dataplane); err != nil {
		logrus.Fatalf("nsm-vpp-dataplane: failure to access nsm socket at %s with error: %+v, exiting...", dataplane, err)
	}
	conn, err := tools.SocketOperationCheck(dataplane)
	if err != nil {
		logrus.Fatalf("nsm-vpp-dataplane: failure to communicate with the socket %s with error: %+v", dataplane, err)
	}
	defer conn.Close()
	logrus.Infof("nsm-vpp-dataplane: connection to dataplane registrar socket %s succeeded.", dataplane)

	dataplaneClient := dataplaneapi.NewDataplaneOperationsClient(conn)

	srcContainerID, err := getContainerID(k8s, *srcPodName, *srcPodNamespace)
	if err != nil {
		logrus.Fatalf("Failed to get container ID for pod %s/%s with error: %+v", *srcPodNamespace, *srcPodName, err)
	}
	dstContainerID, err := getContainerID(k8s, *dstPodName, *dstPodNamespace)
	if err != nil {
		logrus.Fatalf("Failed to get container ID for pod %s/%s with error: %+v", *dstPodNamespace, *dstPodName, err)
	}
	logrus.Infof("Source container id: %s destination container id: %s", srcContainerID, dstContainerID)

	srcPID, err := getPidForContainer(srcContainerID)
	if err != nil {
		logrus.Fatalf("fail getting container %s pid with error: %+v", srcContainerID, err)
	}

	dstPID, err := getPidForContainer(dstContainerID)
	if err != nil {
		logrus.Fatalf("ail getting container %s pid with error: %+v", dstContainerID, err)
	}

	srcNamespace := fmt.Sprintf("pid:%d", srcPID)
	dstNamespace := fmt.Sprintf("pid:%d", dstPID)
	logrus.Infof("Source container namespace: %s destination container namespace: %s", srcNamespace, dstNamespace)

	tests := []dataplaneClientTest{
		{
			testName: "all good",
			localSource: &common.LocalMechanism{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					nsmvpp.NSMkeyNamespace:        srcNamespace,
					nsmvpp.NSMkeyIPv4:             "2.2.2.2",
					nsmvpp.NSMkeyIPv4PrefixLength: "24",
				},
			},
			localDestination: &dataplaneapi.Connection_Local{
				Local: &common.LocalMechanism{
					Type: common.LocalMechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{
						nsmvpp.NSMkeyNamespace:        dstNamespace,
						nsmvpp.NSMkeyIPv4:             "2.2.2.3",
						nsmvpp.NSMkeyIPv4PrefixLength: "24"},
				},
			},
			shouldFail: false,
		},
		{
			testName: "missing source namespace",
			localSource: &common.LocalMechanism{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					// nsmutils.NSMkeyNamespace:        srcNamespace,
					nsmvpp.NSMkeyIPv4:             "2.2.2.2",
					nsmvpp.NSMkeyIPv4PrefixLength: "24",
				},
			},
			localDestination: &dataplaneapi.Connection_Local{
				Local: &common.LocalMechanism{
					Type: common.LocalMechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{
						nsmvpp.NSMkeyNamespace:        dstNamespace,
						nsmvpp.NSMkeyIPv4:             "2.2.2.3",
						nsmvpp.NSMkeyIPv4PrefixLength: "24"},
				},
			},
			shouldFail: true,
		},
		{
			testName: "source has ip, but destination doesn't",
			localSource: &common.LocalMechanism{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					nsmvpp.NSMkeyNamespace:        srcNamespace,
					nsmvpp.NSMkeyIPv4:             "2.2.2.2",
					nsmvpp.NSMkeyIPv4PrefixLength: "24",
				},
			},
			localDestination: &dataplaneapi.Connection_Local{
				Local: &common.LocalMechanism{
					Type: common.LocalMechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{
						nsmvpp.NSMkeyNamespace: dstNamespace,
						// nsmutils.NSMkeyIPv4:             "2.2.2.3",
						nsmvpp.NSMkeyIPv4PrefixLength: "24"},
				},
			},
			shouldFail: true,
		},
		{
			testName: "wrong prefix length",
			localSource: &common.LocalMechanism{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					nsmvpp.NSMkeyNamespace:        srcNamespace,
					nsmvpp.NSMkeyIPv4:             "2.2.2.2",
					nsmvpp.NSMkeyIPv4PrefixLength: "34",
				},
			},
			localDestination: &dataplaneapi.Connection_Local{
				Local: &common.LocalMechanism{
					Type: common.LocalMechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{
						nsmvpp.NSMkeyNamespace:        dstNamespace,
						nsmvpp.NSMkeyIPv4:             "2.2.2.3",
						nsmvpp.NSMkeyIPv4PrefixLength: "24"},
				},
			},
			shouldFail: true,
		},
	}
	for _, test := range tests {
		logrus.Infof("Running test: %s", test.testName)
		reply, err := dataplaneClient.ConnectRequest(context.Background(), &dataplaneapi.Connection{
			LocalSource: test.localSource,
			Destination: test.localDestination,
		})
		if err != nil {
			if !test.shouldFail {
				logrus.Fatalf("Test %s failed with error: %+v but should not.", test.testName, err)
			}
		}
		if err == nil {
			if test.shouldFail {
				logrus.Fatalf("Test %s did not fail but should.", test.testName)
			}
			// Need cleanup interfaces for the next test run
			reply, err = dataplaneClient.DisconnectRequest(context.Background(), &dataplaneapi.Connection{
				ConnectionId: reply.ConnectionId,
				LocalSource:  test.localSource,
				Destination:  test.localDestination,
			})
			if err != nil {
				logrus.Fatalf("Failed to call DisconnectRequest with error: %+v", err)
			}
		}
		logrus.Infof("Test %s has finished with expected result.", test.testName)
	}
}

// Returns the first pid in a container.
// borrowed from docker/utils/utils.go
// modified to only return the first pid
// modified to glob with id
// modified to search for newer docker containers
func getPidForContainer(id string) (int, error) {
	pid := 0

	// memory is chosen randomly, any cgroup used by docker works
	cgroupType := "memory"

	cgroupRoot, err := findCgroupMountpoint(cgroupType)
	if err != nil {
		return pid, err
	}

	cgroupThis, err := getThisCgroup(cgroupType)
	if err != nil {
		return pid, err
	}

	id += "*"

	attempts := []string{
		filepath.Join(cgroupRoot, cgroupThis, id, "tasks"),
		// With more recent lxc versions use, cgroup will be in lxc/
		filepath.Join(cgroupRoot, cgroupThis, "lxc", id, "tasks"),
		// With more recent docker, cgroup will be in docker/
		filepath.Join(cgroupRoot, cgroupThis, "docker", id, "tasks"),
		// Even more recent docker versions under systemd use docker-<id>.scope/
		filepath.Join(cgroupRoot, "system.slice", "docker-"+id+".scope", "tasks"),
		// Even more recent docker versions under cgroup/systemd/docker/<id>/
		filepath.Join(cgroupRoot, "..", "systemd", "docker", id, "tasks"),
		// Kubernetes with docker and CNI is even more different
		filepath.Join(cgroupRoot, "..", "systemd", "kubepods", "*", "pod*", id, "tasks"),
		// Another flavor of containers location in recent kubernetes 1.11+
		filepath.Join(cgroupRoot, cgroupThis, "kubepods.slice", "kubepods-besteffort.slice", "*", "docker-"+id+".scope", "tasks"),
		// When runs inside of a container with recent kubernetes 1.11+
		filepath.Join(cgroupRoot, "kubepods.slice", "kubepods-besteffort.slice", "*", "docker-"+id+".scope", "tasks"),
	}

	var filename string
	for _, attempt := range attempts {
		filenames, _ := filepath.Glob(attempt)
		if len(filenames) > 1 {
			return pid, fmt.Errorf("Ambiguous id supplied: %v", filenames)
		} else if len(filenames) == 1 {
			filename = filenames[0]
			break
		}
	}

	if filename == "" {
		return pid, fmt.Errorf("Unable to find container: %v", id[:len(id)-1])
	}

	output, err := ioutil.ReadFile(filename)
	if err != nil {
		return pid, err
	}

	result := strings.Split(string(output), "\n")
	if len(result) == 0 || len(result[0]) == 0 {
		return pid, fmt.Errorf("No pid found for container")
	}

	pid, err = strconv.Atoi(result[0])
	if err != nil {
		return pid, fmt.Errorf("Invalid pid '%s': %s", result[0], err)
	}

	return pid, nil
}

// borrowed from docker/utils/utils.go
func findCgroupMountpoint(cgroupType string) (string, error) {
	output, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}

	// /proc/mounts has 6 fields per line, one mount per line, e.g.
	// cgroup /sys/fs/cgroup/devices cgroup rw,relatime,devices 0 0
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, " ")
		if len(parts) == 6 && parts[2] == "cgroup" {
			for _, opt := range strings.Split(parts[3], ",") {
				if opt == cgroupType {
					return parts[1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("cgroup mountpoint not found for %s", cgroupType)
}

// Returns the relative path to the cgroup docker is running in.
// borrowed from docker/utils/utils.go
// modified to get the docker pid instead of using /proc/self
func getThisCgroup(cgroupType string) (string, error) {
	dockerpid, err := ioutil.ReadFile("/var/run/docker.pid")
	if err != nil {
		return "", err
	}
	result := strings.Split(string(dockerpid), "\n")
	if len(result) == 0 || len(result[0]) == 0 {
		return "", fmt.Errorf("docker pid not found in /var/run/docker.pid")
	}
	pid, err := strconv.Atoi(result[0])
	if err != nil {
		return "", err
	}
	output, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, ":")
		// any type used by docker should work
		if parts[1] == cgroupType {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("cgroup '%s' not found in /proc/%d/cgroup", cgroupType, pid)
}
