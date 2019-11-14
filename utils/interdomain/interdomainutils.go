package interdomain

import (
	"net"
	"strings"

	"github.com/pkg/errors"
)

// ParseNsmURL parses nsm url of the form nsmName@nsmAddress
func ParseNsmURL(nsmURL string) (nsmName, nsmAddress string, err error) {
	if !strings.Contains(nsmURL, "@") {
		return nsmURL, "", errors.Errorf("cannot parse Network Service Manager URL: %s", nsmURL)
	}

	t := strings.Split(nsmURL, "@")
	return strings.Join(t[:len(t)-1], "@"), t[len(t)-1], nil
}

// ResolveDomain translates network service domain name to an IP address
func ResolveDomain(remoteDomain string) (string, error) {
	ip, err := net.LookupIP(remoteDomain)
	if err != nil {
		return "", err
	}
	return ip[0].String(), nil
}
