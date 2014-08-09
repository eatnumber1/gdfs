package drive

/*
import (
	"time"
	"fmt"
	"log"
	"os"
	"syscall"
	"io"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
)

// TODO: Support partial requests and responses

type File struct {
	file *gdrive.File
	drive *Drive
}

func NewFileFromId(drive *Drive, id string) (*File, error) {

	file, err := drive.Files.Get(id).Do()
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

func (this *File) Mode() (mode os.FileMode, err error) {
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
	return this.Mtime()
}

func (this *File) Crtime() (created time.Time, err error) {
	if this.file.CreatedDate == "" {
		return
	}
	return time.Parse(time.RFC3339, this.file.CreatedDate)
}

func (this *File) Children() ([]*File, error) {
	isDir, err := this.IsDirectory()
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, fuse.Errno(syscall.ENOTDIR)
	}

	call := this.drive.Files.List()
	// TODO: This is an injection!
	call.Q(fmt.Sprintf("'%s' in parents", this.file.Id))
	children, err := call.Do()
	if err != nil {
		return nil, err
	}

	dirs := make([]*File, len(children.Items))

	for idx := range children.Items {
		dirs[idx] = NewFile(this.drive, children.Items[idx])
	}

	return dirs, nil
}

func (this *File) IsDirectory() (ret bool, err error) {
	mode, err := this.Mode()
	if err != nil {
		return
	}

	ret = mode & os.ModeDir != 0
	return
}

func (this *File) ChildByName(name string) (*File, error) {
	isDir, err := this.IsDirectory()
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, fuse.Errno(syscall.ENOTDIR)
	}

	call := this.drive.Files.List()
	// TODO: This is an injection!
	call.Q(fmt.Sprintf("'%s' in parents and title = '%s'", this.file.Id, name))
	children, err := call.Do()
	if err != nil {
		return nil, err
	}

	if len(children.Items) > 1 {
		panic("Multiple files with the same name!")
	} else if len(children.Items) == 0 {
		return nil, fuse.ENOENT
	}

	return NewFile(this.drive, children.Items[0]), nil
}

func (this *File) Name() (string, error) {
	return this.file.Title, nil
}

type OpenFile io.ReadCloser

func (this *File) Open() (OpenFile, error) {
	log.Printf("Fetching url %s\n", this.file.DownloadUrl)
	resp, err := this.drive.client.Get(this.file.DownloadUrl)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case 200:
	case 404:
		return nil, fuse.ENOENT
	default:
		log.Printf("Http error %d: %s\n", resp.StatusCode, resp.Status)
		return nil, fuse.EIO
	}

	return resp.Body, nil
}
*/
