package caddyfile

import (
	"flag"
	"github.com/caddyserver/caddy"
	"os"
)

// ParseCorefilePath parses corefile path from arguments
func ParseCorefilePath() string {
	cl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	path := cl.String("conf", caddy.DefaultConfigFile, "")
	_ = cl.Parse(os.Args[1:])
	return *path
}
