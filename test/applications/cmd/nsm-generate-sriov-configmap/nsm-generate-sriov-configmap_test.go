// +build unit_test

package main

import (
	"testing"
)

// mockVFs will be used for unit testing
func mockVFs() *VFs {

	discoveredVFs := map[string]*VF{
		"0000-81-10.6": {ParentDevice: "enp129s0f0", VFLocalID: 3, PCIAddr: "0000-81-10.6", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "95"},
		"0000-81-11.3": {ParentDevice: "enp129s0f1", VFLocalID: 5, PCIAddr: "0000-81-11.3", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "105"},
		"0000-81-11.7": {ParentDevice: "enp129s0f1", VFLocalID: 7, PCIAddr: "0000-81-11.7", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "107"},
		"0000-81-11.0": {ParentDevice: "enp129s0f0", VFLocalID: 4, PCIAddr: "0000-81-11.0", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "96"},
		"0000-81-10.1": {ParentDevice: "enp129s0f1", VFLocalID: 0, PCIAddr: "0000-81-10.1", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "100"},
		"0000-81-11.2": {ParentDevice: "enp129s0f0", VFLocalID: 5, PCIAddr: "0000-81-11.2", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "97"},
		"0000-81-11.4": {ParentDevice: "enp129s0f0", VFLocalID: 6, PCIAddr: "0000-81-11.4", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "98"},
		"0000-81-11.6": {ParentDevice: "enp129s0f0", VFLocalID: 7, PCIAddr: "0000-81-11.6", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "99"},
		"0000-81-11.1": {ParentDevice: "enp129s0f1", VFLocalID: 4, PCIAddr: "0000-81-11.1", pciVendor: "8086", pciType: "1515", NetworkService: nsmSRIOVDefaultNetworkServiceName, iommuGroup: "104"},
	}
	vfs := VFs{}
	vfs.vfs = discoveredVFs

	return &vfs
}

// Dummy test to keep test file alive
func TestConfigMapParse(t *testing.T) {
	vfs := mockVFs()
	if len(vfs.vfs) != 9 {
		t.Fatalf("failed, expected 9 VFs but got %d", len(vfs.vfs))
	}
}
