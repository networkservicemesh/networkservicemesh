package utils

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestProcessOutputShouldNotLostOutput(t *testing.T) {
	assert := gomega.NewWithT(t)
	const expected = "output..."
	for i := 0; i < 1000; i++ {
		output, err := RunCommand(context.Background(), fmt.Sprintf("echo \"%v\"", expected), "", func(s string) {}, bufio.NewWriter(&strings.Builder{}), nil, nil, true)
		assert.Expect(err).Should(gomega.BeNil())
		assert.Expect(strings.TrimSpace(output)).Should(gomega.Equal(expected))
	}
}
