package tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	. "github.com/onsi/gomega"
	"testing"
)

func TestVariableSubstitutions(t *testing.T) {
	RegisterTestingT(t)

	env := map[string]string{
		"KUBECONFIG": "~/.kube/config",
	}

	args := map[string]string{
		"cluster-name": "idd",
		"provider-name": "name",
		"random": "r1",
		"uuid": "uu-uu",
		"tempdir": "/tmp",
		"zone-selector": "zone",
	}

	var1, err := utils.SubstituteVariable("qwe ${KUBECONFIG} $(uuid) BBB", env, args)
	Expect(err).To(BeNil())
	Expect(var1).To(Equal("qwe ~/.kube/config uu-uu BBB"))

}

func TestParseCommandLine1(t *testing.T) {
	RegisterTestingT(t)

	t.Run("simple", func(t *testing.T) {
		RegisterTestingT(t)
		Expect(utils.ParseCommandLine("a b c")).To(Equal([]string{"a", "b", "c"}))
	})

	t.Run("spaces", func(t *testing.T) {
		RegisterTestingT(t)
		Expect(utils.ParseCommandLine("a\\ b c")).To(Equal([]string{"a b", "c"}))
	})

	t.Run("strings", func(t *testing.T) {
		RegisterTestingT(t)
		Expect(utils.ParseCommandLine("a \"b    \" c")).To(Equal([]string{"a", "b    ", "c"}))
	})

	t.Run("empty_arg", func(t *testing.T) {
		RegisterTestingT(t)
		Expect(utils.ParseCommandLine("a 	-N \"\"" )).To(Equal([]string{"a", "-N", ""}))
	})


}
