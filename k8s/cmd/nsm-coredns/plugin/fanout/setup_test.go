// +build unit_test

package fanout

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/caddyserver/caddy"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		input           string
		shouldErr       bool
		expectedFrom    string
		expectedIgnored []string
		expectedFails   int

		expectedErr string
	}{
		// positive
		{"fanout 127.0.0.1", false, ".", nil, 2, ""},
		{"fanout 127.0.0.1:53", false, ".", nil, 2, ""},
		{"fanout 127.0.0.1:8080", false, ".", nil, 2, ""},
		{"fanout [::1]:53", false, ".", nil, 2, ""},
		{"fanout [2003::1]:53", false, ".", nil, 2, ""},
		// negative
		{"fanout a27.0.0.1", true, "", nil, 0, "not an IP"},
		{"fanout 127.0.0.1 {\nblaatl\n}\n", true, "", nil, 0, "unknown property"},
		{`fanout ::1
		fanout com ::2`, true, "", nil, 0, "plugin"},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		f, err := parseFanout(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: expected error but found %s for input %s", i, err, test.input)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: expected no error but found one for input %s, got: %v", i, test.input, err)
			}

			if !strings.Contains(err.Error(), test.expectedErr) {
				t.Errorf("Test %d: expected error to contain: %v, found error: %v, input: %s", i, test.expectedErr, err, test.input)
			}
		}

		if !test.shouldErr && f.maxFailCount != test.expectedFails {
			t.Errorf("Test %d: expected: %d, got: %d", i, test.expectedFails, f.maxFailCount)
		}
	}
}

func TestSetupResolvconf(t *testing.T) {
	const resolv = "resolv.conf"
	if err := ioutil.WriteFile(resolv,
		[]byte(`nameserver 10.10.255.252
nameserver 10.10.255.253`), 0666); err != nil {
		t.Fatalf("Failed to write resolv.conf file: %s", err)
	}
	defer os.Remove(resolv)

	tests := []struct {
		input         string
		shouldErr     bool
		expectedErr   string
		expectedNames []string
	}{
		{`fanout ` + resolv, false, "", []string{"10.10.255.252:53", "10.10.255.253:53"}},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		f, err := parseFanout(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: expected error but found %s for input %s", i, err, test.input)
			continue
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: expected no error but found one for input %s, got: %v", i, test.input, err)
			}

			if !strings.Contains(err.Error(), test.expectedErr) {
				t.Errorf("Test %d: expected error to contain: %v, found error: %v, input: %s", i, test.expectedErr, err, test.input)
			}
		}

		if !test.shouldErr {
			for j, n := range test.expectedNames {
				addr := f.clients[j].addr
				if n != addr {
					t.Errorf("Test %d, expected %q, got %q", j, n, addr)
				}
			}
		}
		for _, p := range f.clients {
			_ = p.health.Check()
		}
	}
}
