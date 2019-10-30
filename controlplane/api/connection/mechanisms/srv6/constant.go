// Package srv6 - parameters keys of SRV6 remote mechanism
package srv6

const (
	// MECHANISM string
	MECHANISM = "SRV6"

	// Mechanism parameters
	
	// SrcHostIP -  src localsid of mgmt interface
	SrcHostIP          = "src_host_ip"
	// DstHostIP -  dst localsid of mgmt interface
	DstHostIP          = "dst_host_ip"
	// SrcBSID -  src BSID
	SrcBSID            = "src_bsid"
	// SrcLocalSID -  src LocalSID
	SrcLocalSID        = "src_localsid"
	// DstBSID - dst BSID
	DstBSID            = "dst_bsid"
	// DstLocalSID - dst LocalSID
	DstLocalSID        = "dst_localsid"
	// SrcHardwareAddress -  src hw address
	SrcHardwareAddress = "src_hw_addr"
	// DstHardwareAddress - dst hw address
	DstHardwareAddress = "dst_hw_addr"
)
