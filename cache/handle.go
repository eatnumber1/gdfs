package cache

import (
	"os"
	"io"
)

type Handle struct {
	io.ReadWriteCloser
	io.Seeker
	fd *os.File
	file *File
}

func NewHandle(fd *os.File, file *File) (handle *Handle, err error) {
	return &Handle{
		fd: fd,
		file: file,
	}
}

func (this *Handle) Close() (err error) {
	err = this.fd.Close()
	if err != nil {
		return
	}

	err = this.file.unref()
}

func (this *Handle) Seek(offset int64, whence int) (int64, error) {
	return this.fd.Seek(offset, whence)
}

func (this *Handle) Read(p []byte) (int, error) {
	return this.fd.Read(p)
}

func (this *Handle) Write(p []byte) (int, error) {
	return this.fd.Write(p)
}
