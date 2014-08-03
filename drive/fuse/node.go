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

type Node struct {
	fusefs.Node
	file *drive.File

}


