package drive

import (
	"fmt"
	"strings"
	"sort"
	"log"
	"net/http"

	"github.com/eatnumber1/gdfs/util"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"
)

var _ = log.Printf
var _ = strings.Split
var _ = sort.Sort
var _ = fmt.Sprintf
var _ = util.WithHere

type Drive struct {
	*gdrive.Service
	About *gdrive.About
	client *http.Client
}

func NewDrive(svc *gdrive.Service, client *http.Client) (*Drive, error) {
	about, err := svc.About.Get().Do()
	if err != nil {
		return nil, err
	}

	return &Drive{
		Service: svc,
		About: about,
		client: client,
	}, nil
}

func (this *Drive) Root() (*File, error) {
	return NewFileFromId(this, this.About.RootFolderId)
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
