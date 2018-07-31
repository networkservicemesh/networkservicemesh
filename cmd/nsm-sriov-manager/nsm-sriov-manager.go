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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	netDirectory    = "/sys/class/net/"
	pciDirectory    = "/sys/bus/pci/devices"
	sriovCapable    = "/sriov_totalvfs"
	sriovConfigured = "/sriov_numvfs"
	// nsmSRIOVDefaultNetworkServiceName defines a default network service name for all
	// SRIOV VFs found on a host. Further editing might be required to map VF to real
	// Network Services
	nsmSRIOVDefaultNetworkServiceName = "networkservicemesh.io/sriov/nsm-default"
)

// VF describes a single instance of VF
type VF struct {
	NetworkService string `yaml:"networkService" json:"NetworkService"`
	pciAddr        string // `yaml:"pciAddr" json:"pciAddr"`
	ParentDevice   string `yaml:"parentDevice" json:"parentDevice"`
	VFLocalID      int32  `yaml:"vfLocalID" json:"vfLocalID"`
	pciVendor      string // `yaml:"pciVendor" json:"pciVendor"`
	pciType        string // `yaml:"pciType" json:"pciType"`
	iommuGroup     string // `yaml:"iommuGroup" json:"iommuGroup"`
	VFIODevice     string `yaml:"vfioDevice" json:"vfioDevice"`
}

// VFs is map of ALL found VFs on a specific host kyed by PCI address
type VFs struct {
	vfs map[string]*VF
}

func newVFs() *VFs {
	v := &VFs{}
	vfs := map[string]*VF{}
	v.vfs = vfs
	return v
}

// Returns a list of SRIOV capable PF names as string
func getSriovPfList() ([]string, error) {

	sriovNetDevices := []string{}

	netDevices, err := ioutil.ReadDir(netDirectory)
	if err != nil {
		logrus.Errorf("Error. Cannot read %s for network device names. Err: %v", netDirectory, err)
		return sriovNetDevices, err
	}

	if len(netDevices) < 1 {
		logrus.Errorf("Error. No network device found in %s directory", netDirectory)
		return sriovNetDevices, err
	}

	for _, dev := range netDevices {
		sriovFilePath := filepath.Join(netDirectory, dev.Name(), "device", "sriov_numvfs")
		f, err := os.Lstat(sriovFilePath)
		if err == nil {
			if f.Mode().IsRegular() { // and its a regular file
				sriovNetDevices = append(sriovNetDevices, dev.Name())
			}
		}
	}

	return sriovNetDevices, nil
}

func readLinkData(link string) (string, error) {
	dirInfo, err := os.Lstat(link)
	if err != nil {
		return "", fmt.Errorf("Error. Could not get directory information %s with error: %v", link, err)
	}

	if (dirInfo.Mode() & os.ModeSymlink) == 0 {
		return "", fmt.Errorf("Error. No symbolic link %s", link)
	}

	info, err := os.Readlink(link)
	if err != nil {
		return "", fmt.Errorf("Error. Cannot read symbolic %s with error: %+v", link, err)
	}

	return info, nil
}

//Reads DeviceName and gets PCI Addresses of VFs configured
func discoverNetworks(discoveredVFs *VFs) error {

	var pciVendor, pciType string
	// Get a list of SRIOV capable NICs in the host
	v := make(map[string]*VF, 0)
	discoveredVFs.vfs = v
	pfList, err := getSriovPfList()

	if err != nil {
		return err
	}

	if len(pfList) < 1 {
		logrus.Errorf("Error. No SRIOV network device found")
		return fmt.Errorf("Error. No SRIOV network device found")
	}

	for _, dev := range pfList {
		sriovcapablepath := filepath.Join(netDirectory, dev, "device", sriovCapable)
		vfs, err := ioutil.ReadFile(sriovcapablepath)
		if err != nil {
			logrus.Errorf("Error. Could not read sriov_totalvfs in device folder. SRIOV not supported. Err: %v", err)
			return err
		}
		totalvfs := bytes.TrimSpace(vfs)
		numvfs, err := strconv.Atoi(string(totalvfs))
		if err != nil {
			logrus.Errorf("Error. Could not parse sriov_capable file. Err: %v", err)
			return err
		}
		if numvfs > 0 {
			sriovconfiguredpath := netDirectory + dev + "/device" + sriovConfigured
			vfs, err = ioutil.ReadFile(sriovconfiguredpath)
			if err != nil {
				logrus.Errorf("Error. Could not read sriov_numvfs file. SRIOV error. %v", err)
				return err
			}
			configuredVFs := bytes.TrimSpace(vfs)
			numconfiguredvfs, err := strconv.Atoi(string(configuredVFs))
			if err != nil {
				logrus.Errorf("Error. Could not parse sriov_numvfs files. Skipping device. Err: %v", err)
				return err
			}

			//get PCI IDs for VFs
			for vf := 0; vf < numconfiguredvfs; vf++ {
				vfDir := fmt.Sprintf("/sys/class/net/%s/device/virtfn%d", dev, vf)
				pciInfo, err := readLinkData(vfDir)
				if err != nil {
					logrus.Errorf("Error. Cannot read symbolic link between virtual function and PCI - Device: %s, VF: %v. Err: %v", dev, vf, err)
					continue
				}
				pciAddr := pciInfo[len("../"):]
				// Getting PCI related info
				vfPCIPath := path.Join(pciDirectory, pciAddr)
				pciVendorPath := path.Join(vfPCIPath, "vendor")
				pciTypePath := path.Join(vfPCIPath, "device")
				iommuGroupPath := path.Join(vfPCIPath, "iommu_group")

				data, err := ioutil.ReadFile(pciVendorPath)
				if err != nil {
					logrus.Errorf(" Cannot read PCI vendor file for %s, VF %v is %s", dev, vf, pciAddr, err)
					continue
				}
				data = bytes.Trim(data, "\n")
				if strings.HasPrefix(string(data), "0x") {
					pciVendor = strings.Split(string(data), "0x")[1]
				} else {
					pciVendor = string(data)
				}
				data, err = ioutil.ReadFile(pciTypePath)
				if err != nil {
					logrus.Errorf(" Cannot read PCI type file for %s, VF %v is %s", dev, vf, pciAddr, err)
					continue
				}
				data = bytes.Trim(data, "\n")
				if strings.HasPrefix(string(data), "0x") {
					pciType = strings.Split(string(data), "0x")[1]
				} else {
					pciType = string(data)
				}

				iommuGroup, err := readLinkData(iommuGroupPath)
				if err != nil {
					logrus.Errorf("Error. Cannot read symbolic link between virtual function and PCI - Device: %s, VF: %v. Err: %v", dev, vf, err)
					continue
				}
				iommuGroup = strings.Split(iommuGroup, "/")[len(strings.Split(iommuGroup, "/"))-1]
				pciAddr = strings.Replace(pciAddr, ":", "-", -1)
				discoveredVFs.vfs[pciAddr] = &VF{}
				discoveredVFs.vfs[pciAddr].pciAddr = pciAddr
				discoveredVFs.vfs[pciAddr].iommuGroup = iommuGroup
				discoveredVFs.vfs[pciAddr].ParentDevice = dev
				discoveredVFs.vfs[pciAddr].pciType = pciType
				discoveredVFs.vfs[pciAddr].pciVendor = pciVendor
				discoveredVFs.vfs[pciAddr].VFLocalID = int32(vf)
				discoveredVFs.vfs[pciAddr].NetworkService = nsmSRIOVDefaultNetworkServiceName
			}
		}
	}
	return nil
}

func buildSRIOVConfigMap(discoveredVFs *VFs) (v1.ConfigMap, error) {
	configMap := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nsm-sriov-vf-list",
			Namespace: "default",
			Labels:    map[string]string{"networkservicemesh.io/sriov": ""},
		},
		Data: make(map[string]string),
	}
	dataString := make(map[string]string, 0)
	for _, vf := range discoveredVFs.vfs {
		data, err := yaml.Marshal(vf)
		if err != nil {
			return v1.ConfigMap{}, err
		}
		dataString[vf.pciAddr] = string(data)
	}
	configMap.Data = dataString

	return configMap, nil
}

func checkVFIOModule() error {
	// Checking if module vfio-pci is loaded
	out, err := exec.Command("lsmod").Output()
	if err != nil {
		return fmt.Errorf("lsmod failed with error: %+v, generated output: %s", err, string(out))
	}
	if strings.Contains(string(out), "vfio_pci") {
		logrus.Infof("vfio-pci kernel module has been found already loaded.")
		return nil
	}
	// Attempting to load vfio-pci kernel module
	out, err = exec.Command("modprobe", "vfio-pci").Output()
	if err != nil {
		return fmt.Errorf("modprobe failed with error: %+v, generated output: %s", err, string(out))
	}
	return nil
}

func buildVFIODevices(discoveredVFs *VFs) error {
	if err := checkVFIOModule(); err != nil {
		return err
	}

	for i, vf := range discoveredVFs.vfs {
		logrus.Infof("VF: %s %+v", i, vf)
		// Check if there is already vfio device for iommu group of VF
		iommuGroup, err := strconv.Atoi(vf.iommuGroup)
		if err != nil {
			// Something wrong with iommu group for this VF
			delete(discoveredVFs.vfs, vf.pciAddr)
			logrus.Errorf("fail to convert iommu group with error: %+v for pci address: %s",
				err, strings.Replace(vf.pciAddr, "-", ":", -1))
			continue
		}
		vfioDevice := fmt.Sprintf("/dev/vfio/%d", iommuGroup)
		_, err = os.Lstat(vfioDevice)
		if !os.IsNotExist(err) && err != nil {
			// Something wrong with access vfio path for iommu group
			// it is safer to skip it and remove from the list of available VFs
			// on the host.
			delete(discoveredVFs.vfs, vf.pciAddr)
			logrus.Errorf("fail to check for existing vfio device with error: %+v for pci address: %s",
				err, strings.Replace(vf.pciAddr, "-", ":", -1))
			continue
		}
		// vfio device for iommu group does not exist, need to create it
		discoveredVFs.vfs[vf.pciAddr].VFIODevice = vfioDevice
		if err := bindVF(vf); err != nil {
			// Could not bind, cannot use, delete this VF
			logrus.Errorf("fail to bind VF to vfio device with error: %+v for pci address: %s",
				err, strings.Replace(vf.pciAddr, "-", ":", -1))
			delete(discoveredVFs.vfs, vf.pciAddr)
			continue
		}
	}
	return nil
}

func waitAndRetry(timeout time.Duration, retries int, check func() bool) error {
	ticker := time.NewTicker(timeout)
	for r := 0; r < retries; r++ {
		select {
		case <-ticker.C:
			if check() {
				return nil
			}
		}

	}

	return fmt.Errorf("timeout has expired")
}

func bindVF(vf *VF) error {
	pciAddr := strings.Replace(vf.pciAddr, "-", ":", -1)
	unbindPath := fmt.Sprintf("/sys/bus/pci/devices/%s/driver/unbind", pciAddr)
	cmdUnbind := exec.Command("echo", pciAddr)
	u, err := os.OpenFile(unbindPath, os.O_WRONLY, 0200)
	if err != nil {
		return fmt.Errorf("fail to open unbind path %s with error: %+v", unbindPath, err)
	}
	defer u.Close()
	cmdUnbind.Stdout = u
	if err := cmdUnbind.Run(); err != nil {
		return fmt.Errorf("unbind failed with error: %+v", err)
	}

	bindArgs := fmt.Sprintf(" %s %s ", vf.pciVendor, vf.pciType)
	cmdBind := exec.Command("echo", bindArgs)
	bindPath := "/sys/bus/pci/drivers/vfio-pci/new_id"
	b, err := os.OpenFile(bindPath, os.O_WRONLY, 0200)
	if err != nil {
		return fmt.Errorf("fail to open bind path %s with error: %+v", unbindPath, err)
	}
	defer b.Close()
	cmdBind.Stdout = b
	if err := cmdBind.Run(); err != nil {
		return fmt.Errorf("bind failed with error: %+v", err)
	}

	return waitAndRetry(200*time.Millisecond, 5, func() bool {
		_, err := os.Stat(vf.VFIODevice)
		if err == nil {
			return true
		}
		return false
	})
}

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	logrus.Infof("Starting SRIOV Network Device Plugin...")
	discoveredVFs := newVFs()
	if err := discoverNetworks(discoveredVFs); err != nil {
		logrus.Errorf("%+v", err)
		os.Exit(1)
	}

	// Uncomment to mock VFs for unit testing
	// discoveredVFs := mockVFs()

	if len(discoveredVFs.vfs) == 0 {
		logrus.Info("No VF were discovered, exiting...")
		os.Exit(0)
	}

	logrus.Infof("%d VFs were discovered on the host.", len(discoveredVFs.vfs))
	// Building vfio device for each VF
	if err := buildVFIODevices(discoveredVFs); err != nil {
		logrus.Errorf("Failed to build VFIO devices for VFs with error: %+v", err)
		os.Exit(1)
	}
	cf, err := buildSRIOVConfigMap(discoveredVFs)
	if err != nil {
		logrus.Errorf("Failed to build SRIOV config map with error: %+v", err)
		os.Exit(1)
	}

	configMap, err := yaml.Marshal(cf)
	if err != nil {
		logrus.Errorf("Failed to marshal SRIOV config map with error: %+v", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile("nsm-sriov-configmap.yaml", configMap, 0644); err != nil {
		logrus.Errorf("Failed to save  SRIOV config map with error: %+v", err)
		os.Exit(1)
	}
	logrus.Info("sriov configmap for Network service mesh has been saved in nsm-sriov-configmap.yaml")
}

// mockVFs will be used for unit testing
func mockVFs() *VFs {

	discoveredVFs := map[string]*VF{
		"0000-81-10.6": &VF{ParentDevice: "enp129s0f0", VFLocalID: 3, pciAddr: "0000-81-10.6", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "95"},
		"0000-81-11.3": &VF{ParentDevice: "enp129s0f1", VFLocalID: 5, pciAddr: "0000-81-11.3", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "105"},
		"0000-81-11.7": &VF{ParentDevice: "enp129s0f1", VFLocalID: 7, pciAddr: "0000-81-11.7", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "107"},
		"0000-81-11.0": &VF{ParentDevice: "enp129s0f0", VFLocalID: 4, pciAddr: "0000-81-11.0", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "96"},
		"0000-81-10.1": &VF{ParentDevice: "enp129s0f1", VFLocalID: 0, pciAddr: "0000-81-10.1", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "100"},
		"0000-81-11.2": &VF{ParentDevice: "enp129s0f0", VFLocalID: 5, pciAddr: "0000-81-11.2", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "97"},
		"0000-81-11.4": &VF{ParentDevice: "enp129s0f0", VFLocalID: 6, pciAddr: "0000-81-11.4", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "98"},
		"0000-81-11.6": &VF{ParentDevice: "enp129s0f0", VFLocalID: 7, pciAddr: "0000-81-11.6", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "99"},
		"0000-81-11.1": &VF{ParentDevice: "enp129s0f1", VFLocalID: 4, pciAddr: "0000-81-11.1", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "104"},
	}
	vfs := VFs{}
	vfs.vfs = discoveredVFs

	return &vfs
}
