package tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
)

func TestVariableSubstitutions(t *testing.T) {
	g := NewWithT(t)

	env := map[string]string{
		"KUBECONFIG": "~/.kube/config",
	}

	args := map[string]string{
		"cluster-name":  "idd",
		"provider-name": "name",
		"random":        "r1",
		"uuid":          "uu-uu",
		"tempdir":       "/tmp",
		"zone-selector": "zone",
	}

	var1, err := utils.SubstituteVariable("qwe ${KUBECONFIG} $(uuid) BBB", env, args)
	g.Expect(err).To(BeNil())
	g.Expect(var1).To(Equal("qwe ~/.kube/config uu-uu BBB"))

}

func TestParseCommandLine1(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(utils.ParseCommandLine("a b c")).To(Equal([]string{"a", "b", "c"}))
	})

	t.Run("spaces", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(utils.ParseCommandLine("a\\ b c")).To(Equal([]string{"a b", "c"}))
	})

	t.Run("strings", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(utils.ParseCommandLine("a \"b    \" c")).To(Equal([]string{"a", "b    ", "c"}))
	})

	t.Run("empty_arg", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(utils.ParseCommandLine("a 	-N \"\"")).To(Equal([]string{"a", "-N", ""}))
	})
}

func TestParseCommandLine2(t *testing.T) {
	g := NewWithT(t)

	cmdLine := utils.ParseCommandLine("go test ./test/integration -test.timeout 300s -count 1 --run \"^(TestNSMHealLocalDieNSMD)$\" --tags \"basic recover usecase\" --test.v")
	g.Expect(len(cmdLine)).To(Equal(12))
}
func TestParseCommandLine3(t *testing.T) {
	g := NewWithT(t)

	cmdLine := utils.ParseCommandLine("go test ./test/integration -test.timeout 300s -count 1 --run \"^(TestNSMHealLocalDieNSMD)$\\\\z\" --tags \"basic recover usecase\" --test.v")
	g.Expect(len(cmdLine)).To(Equal(12))
	g.Expect(cmdLine[8]).To(Equal("^(TestNSMHealLocalDieNSMD)$\\z"))
}
