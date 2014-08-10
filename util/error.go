package util

import (
	"log"

	fuse "bazil.org/fuse"
)

func FuseErrorOrFatalf(err error) fuse.Error {
	switch err.(type) {
	case fuse.Errno:
		return err
	default:
		log.Fatal(err)
	}
	panic("l'impossible!")
}
