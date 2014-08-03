package gdfs

import (
	"log"
	"bytes"
	"fmt"
	"os"
	"io/ioutil"
	"sync/atomic"

	"github.com/eatnumber1/gdfs/drive"
	"github.com/eatnumber1/gdfs/cache"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

type Node struct {
	fusefs.Node
	file *drive.File
	cache *cache.File

	attr Attr
	lastAttr *fuse.Attr
}

func NewNode(file *drive.File, cache *cache.File) (node *Node, err error) {
	node = &Node{
		file: file,
		cache: cache,
		attr: NewAttr(file),
	}
	return
}

func (this *Node) Attr() fuse.Attr {
	attr, err := this.attr.Get()
	if err == nil {
		log.Printf("Failed to fetch attr: %v", err)
		return *atomic.LoadPointer(&this.lastAttr).(*fuse.Attr)
	}
	atomic.StorePointer(&this.lastAttr, attr)
	return attr
}

func (this *Node) ReadDir(intr fusefs.Intr) ([]fuse.Dirent, fuse.Error) {
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

func (this *Node) Lookup(name string, intr fusefs.Intr) (fusefs.Node, fuse.Error) {
	file, err := this.file.ChildByName(name)
	if err != nil {
		return nil, err
	}

	return NewNode(file)
}

// TODO: Support partial reads
func (this *Node) ReadAll(intr fusefs.Intr) (body []byte, err fuse.Error) {
	fd, err := this.file.Read()
	if err != nil {
		return
	}
	defer func() {
		e := fd.Close()
		if err != nil {
			err = e
		}
	}()

	body, err = ioutil.ReadAll(fd)
	if err != nil {
		body = nil
		return
	}
	return
}

func min(a, b int64) int64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func (this *Node) Write(req *fuse.WriteRequest, resp *fuse.WriteResponse, intr fusefs.Intr) (err fuse.Error) {
	body, err := this.ReadAll(intr)
	if err != nil {
		return err
	}

	// TODO: Handle these conversions correctly.
	dataLen := int64(len(req.Data))
	bodyCap := int64(cap(body))
	bodyLen := int64(len(body))

	var written int

	if bodyCap >= req.Offset + dataLen {
		if bodyLen < req.Offset + dataLen {
			body = body[:req.Offset + dataLen]
		}
		written = copy(body[req.Offset:], req.Data)
		// TODO: Handle this conversion correctly
		if int64(written) != dataLen {
			panic(fmt.Sprintf("Failed to write all the data. Expected %d, wrote %d", len(req.Data), written))
		}
	} else {
		newBody := make([]byte, req.Offset + dataLen)
		written = copy(newBody, body[:min(req.Offset, bodyLen)])
		written += copy(body[req.Offset:], req.Data)
		// TODO: Handle this conversion correctly
		if int64(written) != req.Offset + dataLen {
			panic("Didn't write all the expected bytes")
		}
		body = newBody
	}

	err = this.file.Update(bytes.NewReader(body))
	if err != nil {
		return
	}

	resp.Size = written
	return
}

func (this *Node) Mkdir(req *fuse.MkdirRequest, intr fusefs.Intr) (node fusefs.Node, err fuse.Error) {
	file, err := this.file.InsertDirectory(req.Name)
	if err != nil {
		return
	}
	return NewNode(file)
}

func (this *Node) Remove(req *fuse.RemoveRequest, intr fusefs.Intr) (err fuse.Error) {
	// TODO: Optimize
	child, err := this.file.ChildByName(req.Name)
	if err != nil {
		return
	}
	return child.Delete()
}

// TODO: Properly handle the file mode
func (this *Node) Create(req *fuse.CreateRequest, resp *fuse.CreateResponse, intr fusefs.Intr) (node fusefs.Node, handle fusefs.Handle, err fuse.Error) {
	file, err := this.file.InsertFile(req.Name, nil)
	if err != nil {
		return
	}
	node, err = NewNode(file)
	if err != nil {
		return
	}
	handle = node
	return
}
