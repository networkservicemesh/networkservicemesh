package utilities

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"os"
)

type Plugin struct {
	idempotent.Impl
	Deps
}

func (p *Plugin) Init() error {
	return p.Impl.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	return nil
}

func (p *Plugin) Close() error {
	return p.Impl.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	return nil
}

func (p *Plugin) ReadLinkData(link string) (string, error) {
	dirInfo, err := os.Lstat(link)
	if err != nil {
		return "", errors.Wrap(fmt.Errorf("could not get directory information %s with error: %v", link, err), 0)
	}

	if (dirInfo.Mode() & os.ModeSymlink) == 0 {
		return "", errors.Wrap(fmt.Errorf("no symbolic link %s", link), 0)
	}

	info, err := os.Readlink(link)
	if err != nil {
		return "", errors.Wrap(fmt.Errorf("cannot read symbolic %s with error: %+v", link, err), 0)
	}

	return info, nil
}
