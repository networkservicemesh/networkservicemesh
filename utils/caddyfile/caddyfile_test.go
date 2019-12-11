// +build unit_test

package caddyfile

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/onsi/gomega"
)

func TestCaddyfileSave(t *testing.T) {
	assert := gomega.NewWithT(t)
	const coreFilePath = "Caddyfile"
	c := NewCaddyfile(coreFilePath)
	c.Write("log")
	err := c.Save()
	assert.Expect(err).Should(gomega.BeNil())
	buf, err := ioutil.ReadFile(coreFilePath)
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(string(buf)).Should(gomega.Equal(c.String()))
	assert.Expect(os.Remove(coreFilePath)).Should(gomega.BeNil())
}

func TestCaddyfileRemove(t *testing.T) {
	assert := gomega.NewWithT(t)
	c := NewCaddyfile("caddyfile.txt")
	c.Write("logs")
	c.Remove("logs")
	expected := ""
	actual := c.String()
	assert.Expect(actual).Should(gomega.Equal(expected))
}
func TestCaddyfileBacic(t *testing.T) {
	assert := gomega.NewWithT(t)
	c := NewCaddyfile("caddyfile.txt")
	c.WriteScope(".:53").Write("log")
	expected := `.:53 {
	log
}
`
	actual := c.String()
	assert.Expect(actual).Should(gomega.Equal(expected))
}
func TestCaddyfileBasicInnerScopes(t *testing.T) {
	assert := gomega.NewWithT(t)
	c := NewCaddyfile("caddyfile.txt")
	c.WriteScope(".:53").Write("log").WriteScope("hosts").Write("127.0.0.1 google.com")
	c.WriteScope("domain1").Write("forward . 10.0.0.10:1234")
	expected := `.:53 {
	log
	hosts {
		127.0.0.1 google.com
	}
}
domain1 {
	forward . 10.0.0.10:1234
}
`
	actual := c.String()
	assert.Expect(actual).Should(gomega.Equal(expected))
}
