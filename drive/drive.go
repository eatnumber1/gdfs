package drive

// TODO: Support interrupting requests

import (
	"fmt"
	"strings"
	"sort"
	"log"
	"net/http"
	"unsafe"
	"sync/atomic"

	"github.com/eatnumber1/gdfs/util"
	"github.com/eatnumber1/gdfs/drive/fetched"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

var _ = log.Printf
var _ = strings.Split
var _ = sort.Sort
var _ = fmt.Sprintf
var _ = util.WithHere

type Drive struct {
	service *gdrive.Service
	client *http.Client
	aboutFetcher *fetched.AboutValue
}

// TODO: Construct the service ourselves.
func NewDrive(svc *gdrive.Service, client *http.Client) *Drive {
	return &Drive{
		service: svc,
		client: client,
		aboutFetcher: fetched.NewAboutValue(svc),
	}
}

func (this *Drive) Root() (node fusefs.Node, err fuse.Error) {
	about, err := this.aboutFetcher.About(nil)
	if err != nil {
		err = util.FuseErrorOrFatalf(err)
		return
	}
	node = NewNodeRef(this, about.RootFolderId)
	return
}

func (this *Drive) Statfs(req *fuse.StatfsRequest, resp *fuse.StatfsResponse, intr fusefs.Intr) (err fuse.Error) {
	if req.Node != 1 {
		panic("Unknown node for statfs")
	}

	// TODO: Get a more reasonable implementation
	resp.Blocks = ^uint64(0)
	resp.Bfree = ^uint64(0)
	resp.Bavail = ^uint64(0)
	resp.Files = uint64(0)
	resp.Ffree = ^uint64(0)

	// TODO: What's a reasonable value here?
	resp.Namelen = ^uint32(0)

	// TODO: What's a reasonable value here?
	resp.Bsize = ^uint32(0) // preferred block size
	resp.Frsize = ^uint32(0) // fundamental block size

	return
}

func (this *Drive) GenerateInode(parentInode uint64, name string) uint64 {
	if parentInode == 1 && name == "" {
		node, err := this.Root()
		if err != nil {
			log.Fatalf("GenerateInode(): error fetching root: %v", err)
			return ^uint64(0)
		}
		inode, err := node.(*NodeRef).Inode()
		if err != nil {
			log.Fatalf("GenerateInode(): error fetching inode: %v", err)
			return ^uint64(0)
		}
		return inode
	}
	panic(fmt.Sprintf("GenerateInode(%v, %v)", parentInode, name))
}

type DriveRef struct {
	*Drive
	newDrive func() *Drive
}

func NewDriveRef(svc *gdrive.Service, client *http.Client) *DriveRef {
	newDrive := func() *Drive {
		return NewDrive(svc, client)
	}

	return &DriveRef{ newDrive(), newDrive }
}

func (this *DriveRef) Reset() {
	this.setDrive(this.newDrive())
}

func (this *DriveRef) getDrive() *Drive {
	return ((*Drive)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&this.Drive)))))
}

func (this *DriveRef) setDrive(drive *Drive) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&this.Drive)), unsafe.Pointer(drive))
}

/*
func (this *Drive) FilePathToId(path string) (string, error) {
	util.WithHere(func(fun string, file string, line int) {
		log.Printf("%s:%d %s(path=\"%v\")", file, line, fun, path)
	})
	splitPath := strings.Split(path, "/")
	sort.Sort(sort.Reverse(sort.StringSlice(splitPath)))
	util.WithHere(func(fun string, file string, line int) {
		log.Printf("%s:%d %s: splitPath=\"%v\"", file, line, fun, splitPath)
	})
	return this.filePathToId(splitPath)
}

// Deprecated
func (this *Drive) FileNameToId(name string) (string, error) {
	return this.FilePathToId(name)
}

func (this *Drive) filePathToId(reversePath []string) (string, error) {
	if len(reversePath) == 1 {
		if reversePath[0] != "" {
			panic("Impossible!")
		}
		return "root", nil
	}

	head := reversePath[0]
	tail := reversePath[1:]

	parentId, err := this.filePathToId(tail)
	if err != nil {
		return "", err
	}

	child, err := this.GetChildByName(parentId, head)
	if err != nil {
		return "", err
	}

	return child.Id, nil
}

func (this *Drive) GetChildByName(folderId string, name string) (*gdrive.ChildReference, error) {
	call := this.Children.List(folderId)
	call.Q(fmt.Sprintf("title = %s", name))
	childList, err := call.Do()
	if err != nil {
		return nil, err
	}
	children := childList.Items

	if len(children) > 1 {
		panic(fmt.Sprintf("More than one file found with name %s", name))
	} else if len(children) == 0 {
		return nil, NewDriveError("File not found", NOT_FOUND)
	}

	return children[0], nil
}

func (this *Drive) GetChildFiles(folderId string) (*gdrive.FileList, error) {
	call := this.Files.List()
	call.Q(fmt.Sprintf("%v in parents", folderId))
	return call.Do()
}

func (this *Drive) UidToPermissionId(uid uint32) string {
	// TODO
	return this.About.User.PermissionId
}

func (this *Drive) FileFromPath(path string) (*File, error) {
	return NewFileFromPath(this, path)
}

func FileIdToInode(fileId string) uint64 {
	var inode uint64 = 0
	bytes := []byte(fileId)
	for idx := range bytes {
		inode += uint64(bytes[idx])
	}
	return inode
}

func MimeToType(mime string) (mode uint32, err error) {
	switch mime {
	case "application/vnd.google-apps.folder":
		mode = fuse.S_IFREG
	case "application/vnd.google-apps.file":
		mode = fuse.S_IFDIR
	default:
		err = NewDriveError(fmt.Sprintf("Unknown mime type \"%s\"", mime), UNKNOWN_MIME)
		return
	}
	return
}
*/
