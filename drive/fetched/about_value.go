package fetched

import (
	gdrive "code.google.com/p/google-api-go-client/drive/v2"
	fusefs "bazil.org/fuse/fs"
)

type AboutValue struct {
	Value
}

func NewAboutValue(service *gdrive.Service) *AboutValue {
	fetchFunc := func(intr fusefs.Intr) (interface{}, error) {
		return service.About.Get().Do()
	}

	return &AboutValue{
		NewValue(fetchFunc),
	}
}

func (this *AboutValue) About(intr fusefs.Intr) (file *gdrive.About, err error) {
	f, err := this.Get(intr)
	if err != nil {
		return
	}
	file = f.(*gdrive.About)
	return
}
