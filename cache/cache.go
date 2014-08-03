package cache

import (
	"flag"
	"syscall"
	"os"

	"github.com/eatnumber1/gdfs/drive"
	"github.com/eatnumber1/gdfs/atomic"
)

var (
	CachePath = flag.String("cachedir", "/tmp", "File cache directory")
)

type Cache struct {
	fd *os.File
	refcnt atomic.ReferenceCounter
}

func NewCache() (cache *Cache, err error) {
	fd, err := os.OpenFile(*CachePath, syscall.O_RDONLY, 0000)
	if err != nil {
		return err
	}

	cache = &Cache{
		fd: fd,
		refcnt: atomic.NewReferenceCounter(),
	}
}

func (this *Cache) File(driveFile *drive.File) (cacheFile *File, err error) {
	return NewCacheFile(this, driveFile)
}

func (this *Cache) Close() error {
	return this.unref()
}

func (this *Cache) ref() {
	this.refcnt.ref()
}

func (this *Cache) unref() (err error) {
	if !this.refcnt.unref() {
		err = this.fd.Close()
		this.fd = nil
	}
}

func (this *Cache) Statfs() (st *syscall.Statfs_t, err error) {
	err = syscall.Fstatfs(this.fd.Fd(), st)
	return
}
