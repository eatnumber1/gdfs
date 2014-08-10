package drive

import (
	"unsafe"
	"sync/atomic"

	"github.com/eatnumber1/gdfs/drive/fetched"
	"github.com/eatnumber1/gdfs/util"
)

type DirHandleCache struct {
	refcnt *util.ReferenceCount

	fetcher fetched.DirValue
	dirents unsafe.Pointer // *[]fuse.Dirent
}

func NewDirHandleCache(fetcher fetched.DirValue) *DirHandleCache {
	return &DirHandleCache{
		refcnt: util.NewReferenceCount(),
		fetcher: fetcher,
	}
}

func (this *DirHandleCache) Ref() {
	this.refcnt.Ref()
}

func (this *DirHandleCache) Unref() {
	if this.refcnt.Unref() {
		atomic.StorePointer(&this.dirents, unsafe.Pointer(nil))
		this.fetcher.Forget()
	}
}
