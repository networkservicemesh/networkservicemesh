package corefile

import (
	"fmt"
	"github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"testing"
)

func TestSave(t *testing.T) {
	gomega.RegisterTestingT(t)
	const coreFilePath = "Corefile"
	c := NewCorefile(coreFilePath)
	c.WriteScope(".").Write("log")
	err := c.Save()
	gomega.Expect(err).Should(gomega.BeNil())
	buf, err := ioutil.ReadFile(coreFilePath)
	gomega.Expect(err).Should(gomega.BeNil())
	gomega.Expect(string(buf)).Should(gomega.Equal(c.String()))
	gomega.Expect(os.Remove(coreFilePath)).Should(gomega.BeNil())
}

func TestCorefilePrioritize(t *testing.T) {
	gomega.RegisterTestingT(t)
	c := NewCorefile("corefile.txt")

	c.WriteScope("domain1 domain2").Write("log")
	c.WriteScope("domain1 domain3").Write("log").Prioritize()

	expected := `domain1 domain3 {
	log
}
domain1 domain2 {
	log
}
`
	actual := c.String()
	gomega.Expect(actual).Should(gomega.Equal(expected))
}

func TestCorefileRemove(t *testing.T) {
	gomega.RegisterTestingT(t)
	c := NewCorefile("corefile.txt")
	c.WriteScope(".53:").Write("logs")
	c.Remove(".53:")
	expected := ""
	actual := c.String()
	gomega.Expect(actual).Should(gomega.Equal(expected))
}
func TestCorefileBacic(t *testing.T) {
	gomega.RegisterTestingT(t)
	c := NewCorefile("corefile.txt")
	c.WriteScope(".53:").Write("log")
	expected := `.53: {
	log
}
`
	actual := c.String()
	gomega.Expect(actual).Should(gomega.Equal(expected))
}
func TestCorefileBacicInnerScopes(t *testing.T) {
	gomega.RegisterTestingT(t)
	c := NewCorefile("corefile.txt")
	c.WriteScope(".53:").
		Write("log").
		WriteScope("hosts").
		Write("127.0.0.1 google.com").Up().Up().
		WriteScope("my.google.com").
		Write(fmt.Sprintf("forward . %v", "10.0.0.10:1234"))
	expected := `.53: {
	log
	hosts {
		127.0.0.1 google.com
	}
}
my.google.com {
	forward . 10.0.0.10:1234
}
`
	actual := c.String()
	gomega.Expect(actual).Should(gomega.Equal(expected))
}
