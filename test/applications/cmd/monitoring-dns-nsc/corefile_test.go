package main

import (
	"fmt"
	"github.com/onsi/gomega"
	"testing"
)

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
	c.Save()
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
