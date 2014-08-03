package fuse

import (
	"log"
	"bytes"
	"fmt"
	"os"
	"io/ioutil"
	"sync/atomic"

	"github.com/eatnumber1/gdfs/drive"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

type DriveFileSystem struct {
	fusefs.Fs
	drive *drive.Drive
}

func (DriveFileSystem) Root(drive *drive.Drive) (fusefs.Node, fuse.Error) {
	rootFile, err := drive.Root()
	if err != nil {
		return
	}

	node, err = NewNode(drive, rootFile)
	return
}
