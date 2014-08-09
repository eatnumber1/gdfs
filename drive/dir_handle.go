package drive

import (
	"os"
	"log"
	"unsafe"
	"sync/atomic"

	"github.com/eatnumber1/gdfs/drive/fetched"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

// TODO: Cache the DirValue and dirents in the node.

type DirHandle struct {
	drive *Drive
	fetcher fetched.DirValue
	dirents unsafe.Pointer // *[]fuse.Dirent
}

func NewDirHandleFromFileValue(drive *Drive, fetcher fetched.FileValue) *DirHandle {
	return NewDirHandle(drive, fetched.NewDirValueFromFileValue(fetcher, drive.service))
}

func NewDirHandle(drive *Drive, fetcher fetched.DirValue) *DirHandle {
	return &DirHandle{
		drive: drive,
		fetcher: fetcher,
	}
}

func (this *DirHandle) ReadDir(intr fusefs.Intr) (dirents []fuse.Dirent, err fuse.Error) {
	dp := (*[]fuse.Dirent)(atomic.LoadPointer(&this.dirents))
	if dp != nil {
		dirents = *dp
		return
	}

	children, err := this.fetcher.List(intr)
	if err != nil {
		return
	}

	var validEnts uint = 0
	ents := make([]fuse.Dirent, len(children))
	for i := range children {
		var child fetched.FileValue = children[i]

		var file *gdrive.File
		file, err = child.File(intr)
		if err != nil {
			return
		}

		var about *gdrive.About
		about, err = this.drive.aboutFetcher.About(intr)
		if err != nil {
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

	atomic.StorePointer(&this.dirents, unsafe.Pointer(&dirents))
	return
}

func (this *DirHandle) Flush(req *fuse.FlushRequest, intr fusefs.Intr) (err fuse.Error) {
	return
}

func (this *DirHandle) Release(req *fuse.ReleaseRequest, intr fusefs.Intr) (err fuse.Error) {
	atomic.StorePointer(&this.dirents, unsafe.Pointer(nil))
	this.fetcher.Forget()
	return
}
