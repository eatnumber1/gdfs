package fetched

import (
	gdrive "code.google.com/p/google-api-go-client/drive/v2"
	fusefs "bazil.org/fuse/fs"
)

type FileValue interface {
	Value
	File(fusefs.Intr) (*gdrive.File, error)
}

type FileValueImpl struct {
	Value
}

func NewFileValue(fileId string, service *gdrive.Service) FileValue {
	fetchFunc := func(intr fusefs.Intr) (interface{}, error) {
		return service.Files.Get(fileId).Do()
	}

	return &FileValueImpl{
		NewValue(fetchFunc),
	}
}

func NewFileValueFromFile(file *gdrive.File) FileValue {
	fetchFunc := func(intr fusefs.Intr) (interface{}, error) {
		return file, nil
	}

	return &FileValueImpl{
		NewValue(fetchFunc),
	}
}

func (this *FileValueImpl) File(intr fusefs.Intr) (file *gdrive.File, err error) {
	f, err := this.Get(intr)
	if err != nil {
		return
	}
	file = f.(*gdrive.File)
	return
}
