package gdfs

import (
	"unsafe"
	"sync"
	"sync/atomic"
	"os"

	"github.com/eatnumber1/gdfs/drive"
	"github.com/eatnumber1/gdfs/cache"

	fuse "bazil.org/fuse"
)

const (
	// TODO: Get rid of these
	OWNER uint32 = 168633
	GROUP uint32 = 5000
)

// TODO: Come up with a lockless algo
type Attr struct {
	cacheDirty bool

	// Guards the cacheFile and cacheDirty
	cacheMutex sync.RWMutex

	driveFile *drive.File
	cacheFile *cache.File
}

func NewAttr(driveFile *drive.File, cacheFile *cache.File) Attr {
	// The cache starts dirty.
	return Attr{
		cacheDirty: true,
		driveFile: driveFile,
		cacheFile: cacheFile,
	}
}

func (this Attr) Clear() {
	this.cacheMutex.Lock()
	defer this.cacheMutex.Unlock()

	this.cacheDirty = true
}

func (this Attr) Get() (attr *fuse.Attr, err error) {
	st, dirty, err := this.getCachedReadOnly()
	if err != nil {
		return
	}

	if dirty {
		st, err = this.updateAndGetCached()
		if err != nil {
			return
		}
	}

	attr = statToAttr(st)
	return
}

func (this Attr) updateAndGetCached() (st *syscall.Stat_t, err error) {
	this.cacheMutex.Lock()
	defer this.cacheMutex.Unlock()

	if this.cacheDirty {
		st, err = this.cacheFile.Stat()
		if err != nil {
			return
		}

		atime := this.file.Atime()
		mtime := this.file.Mtime()
		atimespec := timeToTimespec(atime)
		mtimespec := timeToTimespec(mtime)
		if st.Atim != atimespec || st.Mtim != mtimespec {
			err = this.file.Utimes(atime, mtime)
			if err != nil {
				return
			}
			st.Atim = atimespec
			st.Mtim = mtimespec
		}

		mode := file.Mode()
		if st.Mode != mode {
			err = this.file.Chmod(mode)
			if err != nil {
				return
			}
			st.Mode = mode
		}

		// TODO: uid and gid with chown

		this.cacheDirty = false
	} else {
		st, err = this.cacheFile.Stat()
	}
	return
}

func (this Attr) getCachedReadOnly() (st *syscall.Stat_t, dirty bool, err error) {
	this.cacheMutex.RLock()
	defer this.cacheMutex.RUnlock()

	dirty = this.cacheDirty
	if dirty {
		return
	}

	st, err = this.cacheFile.Stat()
	return
}

func statToAttr(st *syscall.Stat_t) *fuse.Attr {
	mtime := timespecToTime(st.Mtime)
	atime := timespecToTime(st.Atime)
	mode := st.Mode

	var nlink uint32 = 1
	if mode & os.ModeDir != 0 {
		nlink++
	}

	return &fuse.Attr{
		Inode: file.Inode(),
		Size: st.Size,
		Blocks: 1, // TODO
		Atime: atime,
		Mtime: mtime,
		Ctime: mtime, // TODO
		Crtime: Time(), // TODO
		Mode: mode,
		Nlink: nlink,
		Uid: OWNER,
		Gid: GROUP,
		Rdev: 0, // TODO
		Flags: 0,
	}
}

func timespecToTime(t syscall.Timespec) time.Time {
	return time.Unix(t.Unix())
}

func timeToTimespec(t time.Time) syscall.Timespec {
	return syscall.NsecToTimespec(t.UnixNano())
}
