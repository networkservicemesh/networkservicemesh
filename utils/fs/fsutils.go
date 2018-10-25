package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
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

// FindProcInode Traverse /proc/<pid>/<suffix> files,
// compare their inodes with inode parameter and returns <pid> if inode matches
// use FindProcInode(xxx, "/ns/net") for example
func FindProcInode(inode uint64, suffix string) (uint64, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return 0, fmt.Errorf("can't read /proc directory: %+v", err)
	}

	for _, f := range files {
		name := f.Name()
		if isDigits(name) {
			tryInode, err := GetInode("/proc/" + name + suffix)
			if err != nil {
				return 0, err
			}
			if tryInode == inode {
				uInt, err := strconv.ParseUint(name, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("expecting integer: %+v", err)
				}
				return uInt, nil
			}
		}
	}

	return 0, nil
}
