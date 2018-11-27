package memif

import (
	"github.com/docker/docker/pkg/mount"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

func DirectConnection(crossConnect *crossconnect.CrossConnect, baseDir string) (*crossconnect.CrossConnect, error) {
	src := crossConnect.GetLocalSource().GetMechanism()
	dst := crossConnect.GetLocalDestination().GetMechanism()

	fullyQualifiedDstSocketFilename := path.Join(baseDir, dst.GetWorkspace(), dst.GetSocketFilename())
	dstSocketDir, dstSocketFilename := path.Split(fullyQualifiedDstSocketFilename)

	fullyQualifiedSrcSocketFilename := path.Join(baseDir, src.GetWorkspace(), src.GetSocketFilename())
	srcSocketDir, srcSocketFilename := path.Split(fullyQualifiedSrcSocketFilename)

	if err := createDirectory(srcSocketDir); err != nil {
		return nil, err
	}

	if err := mount.Mount(dstSocketDir, srcSocketDir, "hard", "bind"); err != nil {
		deleteFolder(srcSocketDir)
		return nil, err
	}

	if srcSocketFilename == dstSocketFilename {
		return crossConnect, nil
	}

	if err := os.Symlink(fullyQualifiedDstSocketFilename, fullyQualifiedSrcSocketFilename); err != nil {
		mount.Unmount(srcSocketDir)
		deleteFolder(srcSocketDir)
		return nil, err
	}

	return crossConnect, nil
}

func createDirectory(path string) error {
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}
	logrus.Infof("Create directory: %s", path)
	return nil
}

func deleteFolder(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	logrus.Infof("Remove directory: %s", path)
	return nil
}
