package fs

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"
	"unicode"
)

func isDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// GetInode returns Inode for file
func GetInode(file string) (uint64, error) {
	fileinfo, err := os.Stat(file)
	if err != nil {
		return 0, fmt.Errorf("error stat file: %+v", err)
	}
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("not a stat_t")
	}
	return stat.Ino, nil
}

// FindFileInProc Traverse /proc/<pid>/<suffix> files,
// compare their inodes with inode parameter and returns file if inode matches
// use FindProcInode(xxx, "/ns/net") for example
func FindFileInProc(inode uint64, suffix string) (string, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return "", fmt.Errorf("can't read /proc directory: %+v", err)
	}

	for _, f := range files {
		name := f.Name()
		if isDigits(name) {
			filename := "/proc/" + name + suffix
			tryInode, err := GetInode(filename)
			if err != nil {
				// Just report into log, do not exit
				logrus.Errorf("Can't find %s Error: %v", filename, err)
				continue
			}
			if tryInode == inode {
				if cmdline, err := GetCmdline(name); err == nil && strings.Contains(cmdline, "pause") {
					return filename, nil
				}
			}
		}
	}

	return "", fmt.Errorf("not found")
}

func GetAllNetNs() ([]uint64, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("can't read /proc directory: %+v", err)
	}
	inodes := make([]uint64, 0, len(files))
	for _, f := range files {
		name := f.Name()
		if isDigits(name) {
			filename := path.Join("/proc", name, "/ns/net")
			inode, err := GetInode(filename)
			if err != nil {
				continue
			}
			inodes = append(inodes, inode)
		}
	}
	return inodes, nil
}

func GetCmdline(pid string) (string, error) {
	data, err := ioutil.ReadFile(path.Join("/proc/", pid, "cmdline"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
