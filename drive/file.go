package drive

import (
	"time"
	"fmt"
	"log"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	"github.com/hanwen/go-fuse/fuse"
)

const (
	// TODO: Get rid of this
	OWNER uint32 = 168633
)

const (
	R_OK = 4
	W_OK = 2
	X_OK = 1
)

// TODO: Support partial requests and responses

type File struct {
	file *gdrive.File
	drive *Drive
}

func NewFileFromPath(drive *Drive, path string) (*File, error) {
	fileId, err := drive.FilePathToId(path)
	if err != nil {
		return nil, err
	}

	file, err := drive.Files.Get(fileId).Do()
	if err != nil {
		return nil, err
	}

	return NewFile(drive, file), nil
}

func NewFile(drive *Drive, file *gdrive.File) *File {
	return &File{
		drive: drive,
		file: file,
	}
}

func (this *File) Size() (uint64, error) {
	return uint64(this.file.FileSize), nil
}

func (this *File) Inode() uint64 {
	var inode uint64 = 0
	bytes := []byte(this.file.Id)
	for idx := range bytes {
		inode += uint64(bytes[idx])
	}
	return inode
}

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

func mimeToType(mime string) (mode uint32, err error) {
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

func (this *File) Mode() (mode uint32, err error) {
	// TODO: Map the file owner to a uid/gid
	mode, err = mimeToType(this.file.MimeType)
	if err != nil {
		return 0, err
	}

	for idx := range this.file.Permissions {
		perm := this.file.Permissions[idx]
		// TODO: Figure out how "anyone" permission works
		if perm.Id == this.drive.About.User.PermissionId {
			mode |= drivePermToFsPerm(perm)
		}
	}

	return
}

func (this *File) Atime() (ret time.Time, err error) {
	var viewedByMe time.Time
	if this.file.LastViewedByMeDate != "" {
		viewedByMe, err = time.Parse(time.RFC3339, this.file.LastViewedByMeDate)
		if err != nil {
			return
		}
	}

	modified, err := this.Mtime()
	if err != nil {
		return
	}

	if viewedByMe.After(modified) {
		ret = viewedByMe
	} else {
		ret = modified
	}

	return
}

func (this *File) Mtime() (modified time.Time, err error) {
	if this.file.ModifiedDate == "" {
		return
	}
	return time.Parse(time.RFC3339, this.file.ModifiedDate)
}

func (this *File) Ctime() (created time.Time, err error) {
	if this.file.CreatedDate == "" {
		return
	}
	return time.Parse(time.RFC3339, this.file.CreatedDate)
}
