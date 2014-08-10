package drive

import (
	"os"
	"log"
	"unsafe"
	"sync/atomic"

	"github.com/eatnumber1/gdfs/util"
	"github.com/eatnumber1/gdfs/drive/fetched"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

type DirHandle struct {
	drive *Drive
	cache *DirHandleCache
}

func NewDirHandleFromFileValue(drive *Drive, fetcher fetched.FileValue, cacheptrptr *unsafe.Pointer) *DirHandle {
	return NewDirHandle(drive, fetched.NewDirValueFromFileValue(fetcher, drive.service), cacheptrptr)
}

func NewDirHandle(drive *Drive, fetcher fetched.DirValue, cacheptrptr *unsafe.Pointer) *DirHandle {
	// This is slightly racy. Meh
	cacheptr := atomic.LoadPointer(cacheptrptr)
	var cache *DirHandleCache
	if cacheptr != nil {
		cache = (*(*HandleCache)(cacheptr)).(*DirHandleCache)
		cache.Ref()
	} else {
		cache = NewDirHandleCache(fetcher)
		var handleCache HandleCache = cache
		old := atomic.SwapPointer(cacheptrptr, unsafe.Pointer(&handleCache))
		if old != nil {
			(*((*HandleCache)(old))).Unref()
		}
		// Ref() for the node
		cache.Ref()
	}

	return &DirHandle{
		drive: drive,
		cache: cache,
	}
}

func (this *DirHandle) ReadDir(intr fusefs.Intr) (dirents []fuse.Dirent, err fuse.Error) {
	dp := (*[]fuse.Dirent)(atomic.LoadPointer(&this.cache.dirents))
	if dp != nil {
		dirents = *dp
		return
	}

	children, err := this.cache.fetcher.List(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	var validEnts uint = 0
	ents := make([]fuse.Dirent, len(children))
	for i := range children {
		var child fetched.FileValue = children[i]

		var file *gdrive.File
		file, err = child.File(intr)
		if err != nil {
			err = util.FuseErrorOrFatalf(err)
			return
		}

		var about *gdrive.About
		about, err = this.drive.aboutFetcher.About(intr)
		if err != nil {
			err = util.FuseErrorOrFatalf(err)
			return
		}

		var m os.FileMode
		m, e := mode(file, about)
		if e != nil {
			switch e := e.(type) {
			case DriveError:
				if e.Code() == BANNED_MIME {
					log.Println(e)
					continue
				}
				return nil, e
			default:
				e = util.FuseErrorOrFatalf(e)
				return nil, e
			}
		}

		ents[validEnts] = fuse.Dirent{
			Inode: inode(file.Id),
			Type: modeToType(m),
			Name: file.Title,
		}
		validEnts++
	}
	if validEnts == 0 {
		dirents = make([]fuse.Dirent, 0)
	} else {
		dirents = ents[:validEnts - 1]
	}

	atomic.StorePointer(&this.cache.dirents, unsafe.Pointer(&dirents))
	return
}

func (this *DirHandle) Flush(req *fuse.FlushRequest, intr fusefs.Intr) (err fuse.Error) {
	return
}

func (this *DirHandle) Release(req *fuse.ReleaseRequest, intr fusefs.Intr) (err fuse.Error) {
	this.cache.Unref()
	return
}
