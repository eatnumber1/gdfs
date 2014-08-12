package drive

import (
	"os"
	"log"
	"fmt"
	"syscall"
	"sync/atomic"
	"unsafe"

	"github.com/eatnumber1/gdfs/util"
	"github.com/eatnumber1/gdfs/drive/fetched"

	//gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

const (
	// TODO: Get rid of this
	OWNER uint32 = 168633
)

const (
	R_OK os.FileMode = 4
	W_OK os.FileMode = 2
	X_OK os.FileMode = 1
)

type Node struct {
	fusefs.NodeRef
	drive *Drive
	fetcher fetched.FileValue
	cache unsafe.Pointer // *HandleCache
}

type CacheStorage interface {
	GetCache() HandleCache
	SetCache(HandleCache)
}

func NewNode(drive *Drive, fileId string) *Node {
	return &Node{
		drive: drive,
		fetcher: fetched.NewFileValue(fileId, drive.service),
	}
}

func (this *Node) Inode() (uint64, error) {
	file, err := this.fetcher.File(nil)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return ^uint64(0), err
	}

	return inode(file.Id), nil
}

func (this *Node) Lookup(name string, intr fusefs.Intr) (node fusefs.Node, err fuse.Error) {
	mode, err := this.mode(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	if !isDirectory(mode) {
		err = fuse.Errno(syscall.ENOTDIR)
		return
	}

	file, err := this.fetcher.File(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	// TODO: Cache this

	// TODO: Should we fetch the files here instead?
	call := this.drive.service.Children.List(file.Id)
	// TODO: This is an injection!
	//call.Q(fmt.Sprintf("'%s' in parents and title = '%s'", file.Id, name))
	call.Q(fmt.Sprintf("title = '%s'", name))
	children, err := call.Do()
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return nil, err
	}

	if len(children.Items) > 1 {
		panic("Multiple files with the same name!")
	} else if len(children.Items) == 0 {
		err = fuse.ENOENT
		return
	}

	return NewNode(this.drive, children.Items[0].Id), nil
}

func (this *Node) Getattr(req *fuse.GetattrRequest, resp *fuse.GetattrResponse, intr fusefs.Intr) (err fuse.Error) {
	file, err := this.fetcher.File(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	if file.FileSize < 0 {
		panic("Negative file size")
	}
	size := uint64(file.FileSize)

	mtime, err := mtime(file)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	atime, err := atime(file)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	crtime, err := crtime(file)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	inode, err := this.Inode()
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	about, err := this.drive.aboutFetcher.About(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	mode, err := mode(file, about)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	var blocks uint64 = 1 // TODO
	var nlinks uint32 = 1 // TODO
	if isDirectory(mode) {
		blocks = 0
		nlinks = 2 // TODO
	}

	resp.Attr = fuse.Attr{
		Inode: inode,
		Size: size,
		Blocks: blocks,
		Atime: atime,
		Mtime: mtime,
		Ctime: mtime, // TODO
		Crtime: crtime,
		Mode: mode,
		Nlink: nlinks,
		Uid: OWNER,
		Gid: 0, // TODO
		Rdev: 0, // TODO
		Flags: 0,
	}
	return
}

func (this *Node) Attr() fuse.Attr {
	response := &fuse.GetattrResponse{}
	err := this.Getattr(&fuse.GetattrRequest{}, response, nil)
	if err != nil {
		log.Fatalf("Attr(): %v", err)
	}

	return response.Attr
}

func (this *Node) GetCache() HandleCache {
	ptr := ((*HandleCache)(atomic.LoadPointer(&this.cache)))
	if ptr == nil {
		return nil
	}
	return *ptr
}

func (this *Node) SetCache(cache HandleCache) {
	atomic.StorePointer(&this.cache, unsafe.Pointer(&cache))
}

// TODO: Properly handle req.Flags and set resp.Flags
func (this *Node) Open(req *fuse.OpenRequest, resp *fuse.OpenResponse, intr fusefs.Intr) (handle fusefs.Handle, err fuse.Error) {
	mode, err := this.mode(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	if !isDirectory(mode) && req.Dir {
		err = fuse.Errno(syscall.ENOTDIR)
		return
	} else if isDirectory(mode) && !req.Dir {
		// TODO: Find a better error
		err = fuse.EIO
		return
	}

	if req.Dir {
		handle = NewDirHandle(this.drive, fetched.NewDirValueFromFileValue(this.fetcher, this.drive.service), this)
	} else {
		err = fuse.EIO
		return
	}
	return
}

func (this *Node) mode(intr fusefs.Intr) (ret os.FileMode, err error) {
	file, err := this.fetcher.File(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	about, err := this.drive.aboutFetcher.About(intr)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}

	return mode(file, about)
}

// -- NodeRef

type NodeRef struct {
	*Node
	newNode func() *Node
}

func NewNodeRef(drive *Drive, fileId string) *NodeRef {
	newNode := func() *Node {
		return NewNode(drive, fileId)
	}
	return &NodeRef{ newNode(), newNode }
}

func (this *NodeRef) Reset() {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&this.Node)), unsafe.Pointer(this.newNode()))
}
