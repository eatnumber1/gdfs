package drive

import (
	"unsafe"

	"github.com/eatnumber1/gdfs/drive/fetched"
)

type DirHandleCache struct {
	fetcher fetched.DirValue
	dirents unsafe.Pointer // *[]fuse.Dirent
}

func NewDirHandleCache(fetcher fetched.DirValue) *DirHandleCache {
	return &DirHandleCache{
		fetcher: fetcher,
	}
}
