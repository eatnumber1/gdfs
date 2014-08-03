package cache

import (
	"os"
	"fmt"
	"syscall"
	"time"

	"github.com/eatnumber1/gdfs/atomic"
	"github.com/eatnumber1/gdfs/drive"
)

type File struct {
	driveFile *drive.File
	refcnt atomic.ReferenceCounter
	fd *os.File
	name string
	cacheDir *Cache
}

func NewFile(cacheDir *Cache, driveFile *drive.File) (cacheFile *File, err error) {
	isDir, err := driveFile.IsDirectory()
	if err != nil {
		return
	}
	if isDir {
		panic(fmt.Sprintf("Attempt to create cache file for directory %s", driveFile.Name()))
	}

	cacheFileName := driveFile.Id()
	// TODO: Revisit these file modes
	cacheFd, err := syscall.Openat(cacheDir.fd.Fd(), cacheFileName, syscall.O_CREATE | syscall.O_RDWR, 0600)
	if err != nil {
		return
	}

	cacheFile = &File{
		refcnt: atomic.NewReferenceCounter(),
		driveFile: driveFile,
		fd: os.NewFile(cacheFd, fmt.Sprintf("%s/%s", cacheDir.fd.Name(), cacheFileName)),
		name: cacheFileName,
		cacheDir: cacheDir,
	}
}

func (this *File) ref() {
	this.refcnt.ref()
}

func (this *File) unref() (err error) {
	if !this.refcnt.unref() {
		err = syscall.Unlinkat(this.cacheDir.fd, this.name)
		if err != nil {
			return
		}

		err = this.fd.Close()
		if err != nil {
			return
		}
		this.fd = nil

		err = this.cacheDir.unref()
		if err != nil {
			return
		}
		this.cacheDir = nil
	}
}

func (this *File) Open() (handle *Handle, err error) {
	this.ref()
	defer this.unref()
	// Now we just need to this.ref() at the end if we're successful.

	nfd, err := syscall.Dup(this.fd.Fd())
	if err != nil {
		return
	}

	fd := os.NewFile(nfd, this.fd.Name())

	handle, err = NewHandle(fd, this)
	if err != nil {
		fd.Close()
		return
	}

	this.ref()
}

func (this *File) Forget() error {
	return this.unref()
}

func (this *File) Stat() (st *syscall.Stat_t, err error) {
	err = syscall.Fstat(this.fd.Fd(), st)
	return
}

// TODO: Store the crtime in an xattr
// TODO: Figure out what to do for ctime

func (this *File) Utimes(atime time.Time, mtime time.Time) (err error) {
	timevals := []syscall.Timeval{
		syscall.NsecToTimespec(atime.UnixNano()),
		syscall.NsecToTimespec(mtime.UnixNano()),
	}
	err = syscall.Futimes(this.fd.Fd(), timevals)
	return
}

func (this *File) Chmod(mode uint32) error {
	return syscall.Fchmod(this.fd.Fd(), mode)
}

func (this *File) Chown(uid, gid int) error {
	return syscall.Fchown(this.fd.Fd(), uid, gid)
}
