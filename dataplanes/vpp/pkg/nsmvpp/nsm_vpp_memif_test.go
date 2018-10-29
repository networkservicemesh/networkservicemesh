package nsmvpp

import (
	. "github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	. "github.com/onsi/gomega"
	"testing"
)

func TestNegotiateAllParametersSpecified(t *testing.T) {
	RegisterTestingT(t)
	const socketFile = "memif.sock"

	src := map[string]string{
		NSMSocketFile: socketFile,
		NSMMaster:     "true",
	}

	dst := map[string]string{
		NSMSocketFile: socketFile,
		NSMSlave:      "true",
	}

	negotiateParameters(src, dst)

	Expect(src[NSMSocketFile]).To(Equal(socketFile))
	Expect(src[NSMMaster]).To(Equal("true"))
	Expect(src[NSMSlave]).To(Equal(""))

	Expect(dst[NSMSocketFile]).To(Equal(socketFile))
	Expect(dst[NSMSlave]).To(Equal("true"))
	Expect(dst[NSMMaster]).To(Equal(""))
}

func TestNegotiateSocketNotSpecified(t *testing.T) {
	RegisterTestingT(t)

	//both not specified
	src := map[string]string{
		NSMMaster: "true",
	}

	dst := map[string]string{
		NSMSlave: "true",
	}

	negotiateParameters(src, dst)
	Expect(src[NSMSocketFile]).To(Equal(dst[NSMSocketFile]))
	Expect(src[NSMSocketFile]).To(Equal(DefaultSocketFile))

}

func TestNegotiateOneSocketSpecified(t *testing.T) {
	RegisterTestingT(t)

	src := map[string]string{
		NSMMaster: "true",
	}

	const specifiedSocketName = "specified_name.sock"
	dst := map[string]string{
		NSMSocketFile: specifiedSocketName,
		NSMSlave:      "true",
	}

	negotiateParameters(src, dst)
	Expect(src[NSMSocketFile]).To(Equal(dst[NSMSocketFile]))
	Expect(src[NSMSocketFile]).To(Equal(specifiedSocketName))
	Expect(dst[NSMSocketFile]).To(Equal(specifiedSocketName))
}

func TestNegotiateDifferentSocketSpecified(t *testing.T) {
	RegisterTestingT(t)

	const srcSocket = "src.sock"
	const dstSocket = "dst.sock"

	src := map[string]string{
		NSMSocketFile: srcSocket,
		NSMMaster:     "true",
	}

	dst := map[string]string{
		NSMSocketFile: dstSocket,
		NSMSlave:      "true",
	}
	negotiateParameters(src, dst)
	Expect(src[NSMSocketFile]).To(Equal(srcSocket))
	Expect(dst[NSMSocketFile]).To(Equal(dstSocket))
}

func TestNegotiateRolesNotSpecified(t *testing.T) {
	RegisterTestingT(t)

	src := map[string]string{
		NSMSocketFile: "memif.sock",
	}

	dst := map[string]string{
		NSMSocketFile: "memif.sock",
	}
	negotiateParameters(src, dst)
	Expect(src[NSMMaster]).To(Equal("true"))
	Expect(dst[NSMSlave]).To(Equal("true"))
}

func TestNegotiateRolesOneSpecified(t *testing.T) {
	RegisterTestingT(t)

	src := map[string]string{
		NSMSocketFile: "memif.sock",
		NSMSlave:      "true",
	}

	dst := map[string]string{
		NSMSocketFile: "memif.sock",
	}
	negotiateParameters(src, dst)
	Expect(src[NSMSlave]).To(Equal("true"))
	Expect(dst[NSMMaster]).To(Equal("true"))
}

func TestNegotiateSameRolesSpecified(t *testing.T) {
	RegisterTestingT(t)

	src := map[string]string{
		NSMSocketFile: "memif.sock",
		NSMSlave:      "true",
	}

	dst := map[string]string{
		NSMSocketFile: "memif.sock",
		NSMSlave:      "true",
	}

	err := negotiateParameters(src, dst)
	Expect(err).NotTo(Equal(""))
}

func TestValidation(t *testing.T) {
	RegisterTestingT(t)

	src := map[string]string{
		NSMSocketFile:      "memif.sock",
		NSMSlave:           "not bool",
		NSMPerPodDirectory: "pod_dir",
	}
	err := validateMemif(src)
	Expect(err.Error()).To(ContainSubstring("ParseBool"))

	src = map[string]string{
		NSMSocketFile:      "memif.sock",
		NSMSlave:           "true",
		NSMMaster:          "true",
		NSMPerPodDirectory: "pod_dir",
	}
	err = validateMemif(src)
	Expect(err.Error()).To(ContainSubstring("both master and slave parameter specified"))
}
