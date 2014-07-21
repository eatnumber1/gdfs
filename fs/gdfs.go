package gdfs

import (
	"log"
	"time"
	"fmt"
	"os"

	"github.com/eatnumber1/gdfs/drive"
	"github.com/eatnumber1/gdfs/util"
	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

var _ = log.Printf
var _ = time.Unix
var _ = fmt.Sprintf
var _ = util.WithHere

const (
	// TODO: Get rid of this
	OWNER uint32 = 168633
)

const (
	R_OK = 4
	W_OK = 2
	X_OK = 1
)

type DriveFileSystem struct {
	drive *drive.Drive
}

func NewDriveFileSystem(svc *gdrive.Service) (*DriveFileSystem, error) {
	drive, err := drive.NewDrive(svc)
	if err != nil {
		return nil, err
	}

	return &DriveFileSystem{
		drive: drive,
	}, nil
}

func (this *DriveFileSystem) Root() (node fusefs.Node, err fuse.Error) {
	rootFile, err := this.drive.Root()
	if err != nil {
		return
	}

	node, err = NewNode(rootFile)
	return
}

func (this *DriveFileSystem) Statfs(req *fuse.StatfsRequest, resp *fuse.StatfsResponse, intr fusefs.Intr) fuse.Error {
	if req.Node != 1 {
		panic("Unknown node for statfs")
	}

	// TODO: Get a more reasonable implementation
	resp.Blocks = ^uint64(0)
	resp.Bfree = ^uint64(0)
	resp.Bavail = ^uint64(0)
	resp.Files = uint64(0)
	resp.Ffree = ^uint64(0)
	resp.Bsize = ^uint32(0)
	resp.Namelen = ^uint32(0)
	/*resp.Frsize: ?*/

	return nil
}

func (this *DriveFileSystem) GenerateInode(parentInode uint64, name string) uint64 {
	// TODO: Implement an inode cache
	if parentInode == 1 && name == "" {
		node, err := this.Root()
		if err != nil {
			util.WithHere(func(fun string, file string, line int) {
				log.Printf("%s:%d %s: %v", file, line, fun, err)
			})
			return ^uint64(0)
		}
		return node.(*Node).file.Inode()
	}
	panic(fmt.Sprintf("GenerateInode(%v, %v)", parentInode, name))
}

type Node struct {
	fusefs.Node
	file *drive.File
	attr fuse.Attr
}

func NewNode(file *drive.File) (node *Node, err error) {
	atime, err := file.Atime()
	if err != nil {
		return
	}

	mtime, err := file.Mtime()
	if err != nil {
		return
	}

	ctime, err := file.Ctime()
	if err != nil {
		return
	}

	crtime, err := file.Crtime()
	if err != nil {
		return
	}

	mode, err := file.Mode()
	if err != nil {
		return
	}

	isDir, err := file.IsDirectory()
	if err != nil {
		return
	}

	var nlink uint32 = 1
	if isDir {
		nlink++
	}

	size, err := file.Size()
	if err != nil {
		return
	}

	node = &Node{
		file: file,
		attr: fuse.Attr{
			Inode: file.Inode(),
			Size: size,
			Blocks: 1, // TODO
			Atime: atime,
			Mtime: mtime,
			Ctime: ctime,
			Crtime: crtime,
			Mode: mode,
			Nlink: 2,
			Uid: OWNER,
			Gid: 0,
			Rdev: 0, // TODO
			Flags: 0,
		},
	}
	return
}

func (this *Node) Attr() fuse.Attr {
	return this.attr
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
	dirents = dirents[:validDirents - 1]

	return dirents, nil
}

func (this *Node) Lookup(name string, intr fusefs.Intr) (fusefs.Node, fuse.Error) {
	file, err := this.file.ChildByName(name)
	if err != nil {
		return nil, err
	}

	return NewNode(file)
}

/*

// Deprecated
func (this *Gdfs) getAccessBits(name string, uid uint32) (allowedMode uint32, err error) {
	fileId, err := this.drive.FileNameToId(name)
	if err != nil {
		return 0, err
	}

	perms, err := this.drive.Permissions.List(fileId).Do()
	if err != nil {
		return 0, err
	}

	userPermId := this.drive.UidToPermissionId(uid)

	for idx := range perms.Items {
		perm := perms.Items[idx]
		// TODO: Figure out how "anyone" permission works
		if perm.Id == userPermId {
			allowedMode |= drivePermToFsPerm(perm)
		}
	}

	//allowedMode := drivePermToFsPerm(userPerm) | drivePermToFsPerm(anyonePerm)

	return
}

// Deprecated
func drivePermToFsPerm(perm *gdrive.Permission) (mode uint32) {
	mode = 0
	switch perm.Role {
	case "owner":
		fallthrough
	case "writer":
		mode |= W_OK
		fallthrough
	case "reader":
		mode |= R_OK | X_OK
	default:
		panic(fmt.Sprintf("Unknown role \"%v\"", perm.Role))
	}

	switch perm.Type {
	case "user":
		// offset is zero, so do nothing
	case "anyone":
		mode = (mode << 3) | (mode << 6)
	case "domain":
		fallthrough
	case "group":
		// TODO: Map domain and group to ACLs
		mode = 0
	default:
		panic(fmt.Sprintf("Unknown permission type \"%v\"", perm.Type))
	}

	mode = mode | (mode << 3) | (mode << 6)
	log.Printf("mode = %b\n", mode)
	return
}

func (Gdfs) OnUnmount() {
	log.Println("OnUnmount")
}

func (this *Gdfs) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	log.Printf("OpenDir(name=\"%s\")\n", name)
	//return make([]fuse.DirEntry, 0), fuse.OK

	folderId, err := this.drive.FileNameToId(name)
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	children, err := this.drive.GetChildFiles(folderId)
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	dirs := make([]fuse.DirEntry, len(children.Items))

	for idx := range children.Items {
		file := children.Items[idx]

		mode, err := drive.MimeToType(file.MimeType)
		if err != nil {
			log.Printf("For file \"%s\": %v\n", file.Title, err)
			continue
		}

		dirs[idx] = fuse.DirEntry{
			Name: file.Title,
			Mode: mode,
		}
	}

	return dirs, fuse.OK
}

func (this *Gdfs) GetAttr(name string, context *fuse.Context) (a *fuse.Attr, code fuse.Status) {
	log.Printf("GetAttr(name=\"%s\")\n", name)

	file, err := this.drive.FileFromPath(name)
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	atime, err := file.Atime()
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	mtime, err := file.Mtime()
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	ctime, err := file.Ctime()
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	mode, err := file.Mode()
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	size, err := file.Size()
	if err != nil {
		util.WithHere(func(fun string, file string, line int) {
			log.Printf("%s:%d %s: %v", file, line, fun, err)
		})
		return nil, fuse.EIO
	}

	// https://github.com/hanwen/go-fuse/blob/4d73e177ce5784041e103f6979d62da552d2b8c7/fuse/types_darwin.go#L3-20
	attr := &fuse.Attr{
		Ino: file.Inode(),
		Size: size,
		Blocks: 1,
		Atime: uint64(atime.Unix()),
		Mtime: uint64(mtime.Unix()),
		Ctime: uint64(ctime.Unix()),
		Atimensec: uint32(atime.UnixNano()),
		Mtimensec: uint32(mtime.UnixNano()),
		Ctimensec: uint32(ctime.UnixNano()),
		Mode: mode,
		Nlink: 1,
		Owner: fuse.Owner{
			Uid: OWNER,
			//Gid:,
		},
	}

	return attr, fuse.OK
}

func (Gdfs) Open(name string, flags uint32, context *fuse.Context) (fuseFile nodefs.File, status fuse.Status) {
	log.Println("Open")
	return nil, fuse.EIO
}

func (Gdfs) Chmod(path string, mode uint32, context *fuse.Context) (code fuse.Status) {
	log.Println("Chmod")
	return fuse.EIO
}

func (Gdfs) Chown(path string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	log.Println("Chown")
	return fuse.EIO
}

func (Gdfs) Truncate(path string, offset uint64, context *fuse.Context) (code fuse.Status) {
	log.Println("Truncate")
	return fuse.EIO
}

func (Gdfs) Utimens(path string, Atime *time.Time, Mtime *time.Time, context *fuse.Context) (code fuse.Status) {
	log.Println("Utimens")
	return fuse.EIO
}

func (Gdfs) Readlink(name string, context *fuse.Context) (out string, code fuse.Status) {
	log.Println("Readlink")
	return "not implemented", fuse.EIO
}

func (Gdfs) Mknod(name string, mode uint32, dev uint32, context *fuse.Context) (code fuse.Status) {
	log.Println("Mknod")
	return fuse.EIO
}

func (Gdfs) Mkdir(path string, mode uint32, context *fuse.Context) (code fuse.Status) {
	log.Println("Mkdir")
	return fuse.EIO
}

func (Gdfs) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	log.Println("Unlink")
	return fuse.EIO
}

func (Gdfs) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	log.Println("Rmdir")
	return fuse.EIO
}

func (Gdfs) Symlink(pointedTo string, linkName string, context *fuse.Context) (code fuse.Status) {
	log.Println("Symlink")
	return fuse.EIO
}

func (Gdfs) Rename(oldPath string, newPath string, context *fuse.Context) (code fuse.Status) {
	log.Println("Rename")
	return fuse.EIO
}

func (Gdfs) Link(orig string, newName string, context *fuse.Context) (code fuse.Status) {
	log.Println("Link")
	return fuse.EIO
}

// The permissions model works as follows:
//
// User permissions are mapped to the user bits
// Anyone permissions are mapped to the group and other bits
// Group permissions are mapped to ACLs
// Domain permissions are mapped to ACLs
//
// Only user permissions are implemented.
func (this *Gdfs) Access(name string, mode uint32, context *fuse.Context) fuse.Status {
	log.Printf("Access(name=\"%s\")\n", name)

	allowedMode, err := this.getAccessBits(name, context.Uid)
	if err != nil {
		log.Printf("Access(): %v\n", err)
		return fuse.EIO
	}

	// Convert the three-bit mode to a nine-bit copy of it to represent the full ugo
	mode |= (mode << 3) | (mode << 6)

	if allowedMode & mode == mode {
		return fuse.OK
	} else {
		return fuse.EACCES
	}
}

func (Gdfs) Create(path string, flags uint32, mode uint32, context *fuse.Context) (fuseFile nodefs.File, code fuse.Status) {
	log.Println("Create")
	return nil, fuse.EIO
}

func (Gdfs) StatFs(name string) *fuse.StatfsOut {
	log.Printf("StatFs(name=\"%s\")\n", name)

	// TODO: Get a more reasonable implementation
	return &fuse.StatfsOut{
		Blocks: ^uint64(0),
		Bsize: ^uint32(0),
		Bfree: ^uint64(0),
		Bavail: ^uint64(0),
		Ffree: ^uint64(0),
		NameLen: ^uint32(0),
	}
}
*/
