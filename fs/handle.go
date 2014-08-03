package gdfs

import (
	"log"
	"bytes"
	"fmt"
	"os"
	"io/ioutil"
	"net/http"

	"github.com/eatnumber1/gdfs/drive"
	"github.com/eatnumber1/gdfs/cache"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

type Handle struct {
	fusefs.Handle
	file *drive.File
	cache *cache.Handle
}

func (this *Handle) NewHandle(file *drive.File) (*Handle, error) {
}

func (this *Handle) ReadDir(intr fusefs.Intr) ([]fuse.Dirent, fuse.Error) {
	// TODO: Do something with intr
	children, err := this.file.Children()
	if err != nil {
		return nil, err
	}

	dirents := make([]fuse.Dirent, len(children))
	var validDirents int = 0
	for idx := range children {
		child := children[idx]

		mode, err := child.Mode()
		if err != nil {
			switch e := err.(type) {
			case drive.DriveError:
				if e.Code() == drive.BANNED_MIME {
					log.Println(err)
					continue
				}
				return nil, err
			default:
				return nil, err
			}
		}

		var direntType fuse.DirentType
		if mode & os.ModeDir == 0 {
			direntType = fuse.DT_Dir
		} else {
			direntType = fuse.DT_File
		}

		name, err := child.Name()
		if err != nil {
			return nil, err
		}

		dirents[validDirents] = fuse.Dirent{
			Inode: child.Inode(),
			Type: direntType,
			Name: name,
		}
		validDirents++
	}
	log.Printf("dirents = %v", dirents)
	if validDirents == 0 {
		dirents = make([]fuse.Dirent, 0)
	}

	return dirents, nil
}
