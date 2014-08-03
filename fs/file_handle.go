package gdfs

/*
type FileHandle struct {
	io.ReadWriteCloser
	refcnt uint64
}

func NewCacheFile() (*CacheFile, error) {
	return &CacheFile{
		refcnt: 1
	};
}

func (this *CacheFile) ref() {
	atomic.AddInt64(&this.refcnt, 1)
}

func (this *CacheFile) unref() {
	if atomic.AddInt64(&this.refcnt, ^uint64(0)) == 0 {
		this.file.Close()
	}
}
*/
