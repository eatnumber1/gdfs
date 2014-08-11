package fetched

import (
	"fmt"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"
	fusefs "bazil.org/fuse/fs"
)

type DirValue interface {
	FileValue
	List(fusefs.Intr) ([]FileValue, error)
}

type DirValueImpl struct {
	FileValue
	contents Value
}

func NewDirValue(dirId string, service *gdrive.Service) DirValue {
	return NewDirValueFromFileValue(NewFileValue(dirId, service), service)
}

func NewDirValueFromFileValue(fileValue FileValue, service *gdrive.Service) DirValue {
	fetchFunc := func(intr fusefs.Intr) (out interface{}, err error) {
		var file *gdrive.File
		file, err = fileValue.File(intr)
		if err != nil {
			return
		}

		call := service.Files.List()
		// TODO: This is an injection!
		call.Q(fmt.Sprintf("'%s' in parents", file.Id))
		children, err := call.Do()
		if err != nil {
			return
		}

		dirs := make([]FileValue, len(children.Items))
		for idx := range children.Items {
			dirs[idx] = NewFileValueFromFile(children.Items[idx])
		}

		out = dirs
		return
	}

	return &DirValueImpl{
		fileValue,
		NewValue(fetchFunc),
	}
}

// TODO: Support Lookup

func (this *DirValueImpl) List(intr fusefs.Intr) (list []FileValue, err error) {
	l, err := this.contents.Get(intr)
	if err != nil {
		return
	}
	list = l.([]FileValue)
	return
}
