package common

type InodeSet struct {
	inodes map[uint64]bool
}

func NewInodeSet(inodes []uint64) *InodeSet {
	set := map[uint64]bool{}
	for _, inode := range inodes {
		set[inode] = true
	}
	return &InodeSet{inodes: set}
}

func (i *InodeSet) Contains(inode uint64) bool {
	_, contains := i.inodes[inode]
	return contains
}
