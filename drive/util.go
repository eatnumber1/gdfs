package drive

import (
	"time"
	"os"
	"log"
	"fmt"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
)

func atime(file *gdrive.File) (atime time.Time, err error) {
	if file.LastViewedByMeDate != "" {
		atime, err = time.Parse(time.RFC3339, file.LastViewedByMeDate)
		if err != nil {
			return
		}
	}

	mtime, err := mtime(file)
	if err != nil {
		return
	}

	if atime.IsZero() && mtime.IsZero() {
		crtime, e := crtime(file)
		if e != nil {
			err = e
			return
		}

		atime = crtime
	} else {
		if atime.Before(mtime) {
			atime = mtime
		}
	}

	return
}

func mtime(file *gdrive.File) (modified time.Time, err error) {
	if file.ModifiedDate == "" {
		return
	}
	return time.Parse(time.RFC3339, file.ModifiedDate)
}

func crtime(file *gdrive.File) (created time.Time, err error) {
	if file.CreatedDate == "" {
		return
	}
	return time.Parse(time.RFC3339, file.CreatedDate)
}

func mode(file *gdrive.File, about *gdrive.About) (mode os.FileMode, err error) {
	// TODO: Map the file owner to a uid/gid
	mode, err = mimeToType(file.MimeType)
	if err != nil {
		return 0, err
	}

	for idx := range file.Permissions {
		perm := file.Permissions[idx]
		// TODO: Figure out how "anyone" permission works
		if perm.Id == about.User.PermissionId {
			mode |= drivePermToFsPerm(perm)
		}
	}

	return
}

func drivePermToFsPerm(perm *gdrive.Permission) (mode os.FileMode) {
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
	default:
		panic(fmt.Sprintf("Unknown permission type \"%v\"", perm.Type))
	}

	log.Printf("mode = %b\n", mode)
	return
}

func mimeToType(mime string) (mode os.FileMode, err error) {
	switch mime {
	case "application/vnd.google-apps.folder":
		mode = os.ModeDir
	case "application/vnd.google-apps.document":
		fallthrough
	case "application/vnd.google-apps.drawing":
		fallthrough
	case "application/vnd.google-apps.form":
		fallthrough
	case "application/vnd.google-apps.fusiontable":
		fallthrough
	case "application/vnd.google-apps.presentation":
		fallthrough
	case "application/vnd.google-apps.sites":
		fallthrough
	case "application/vnd.google-apps.script":
		fallthrough
	case "application/vnd.google-apps.spreadsheet":
		err = NewDriveError(fmt.Sprintf("Banned mime type \"%s\"", mime), BANNED_MIME)
		return
	default:
		// Empty
	}
	return
}

func isDirectory(mode os.FileMode) bool {
	return mode & os.ModeDir != 0
}

func inode(id string) uint64 {
	var inode uint64 = 0
	bytes := []byte(id)
	for idx := range bytes {
		inode += uint64(bytes[idx])
	}
	return inode
}

func modeToType(mode os.FileMode) (direntType fuse.DirentType) {
	if mode & os.ModeDir == 0 {
		direntType = fuse.DT_Dir
	} else {
		direntType = fuse.DT_File
	}
	return
}
