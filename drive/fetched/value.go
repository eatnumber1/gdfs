package fetched

import (
	"sync/atomic"
	"sync"
	"runtime"

	fuse "bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

type FetchFunc func(fusefs.Intr) (interface{}, error)

type Value interface {
	Get(fusefs.Intr) (interface{}, error)
}

type valueImpl struct {
	waiters int64
	intr fusefs.Intr
	shutdown chan struct{}
	result chan fetcherResult

	isShutdown int32 // bool

	// Only access using atomics
	fetchOnce *sync.Once
	fetchOnceLock sync.RWMutex
	fetchFunc FetchFunc
}

// This is a container type upon which we place a finalizer. This is because we
// start goroutines which hold pointers to valueImpl which prevents garbage
// collection. Using valueRef's finalizer, we can know to shutdown valueImpl's
// goroutines when valueRef is finalized.
type valueRef struct {
	*valueImpl
}

type fetcherResult struct {
	value interface{}
	err error
}

func (this *valueImpl) Get(intr fusefs.Intr) (interface{}, error) {
	this.beginFetch()
	this.ref()
	defer this.unref()
	select {
	case res := <-this.result:
		return res.value, res.err
	case <-intr:
		return nil, fuse.EINTR
	}
}

func (this *valueImpl) done() {
	close(this.shutdown)
}

func finalizeRef(ref *valueRef) {
	ref.done()
}

func NewValue(fetchFunc FetchFunc) Value {
	impl := &valueImpl{
		waiters: 0,
		intr: make(fusefs.Intr),
		shutdown: make(chan struct{}),
		result: make(chan fetcherResult),
		fetchFunc: fetchFunc,
		fetchOnce: &sync.Once{},
		isShutdown: 0,
	}
	ref := &valueRef{ impl }
	runtime.SetFinalizer(ref, finalizeRef)
	return ref
}

func (this *valueImpl) fetch() {
	intrc := make(fusefs.Intr)
	resc := make(chan fetcherResult, 1)
	go func() {
		value, err := this.fetchFunc(intrc)
		res := fetcherResult{
			value: value,
			err: err,
		}
		resc <- res
	}()

	var res fetcherResult

	select {
	case <-this.intr:
		this.resetFetchOnce()
		close(intrc)
		return
	case <-this.shutdown:
		close(intrc)
		return
	case res = <-resc:
	}

	for {
		select {
		case <-this.shutdown:
			return
		case this.result <- res:
		}
	}
}

func (this *valueImpl) ref() {
	atomic.AddInt64(&this.waiters, int64(1))
}

func (this *valueImpl) unref() {
	refcnt := atomic.AddInt64(&this.waiters, int64(-1))
	if refcnt == -1 {
		panic("unref() called on value with zero waiters")
	// TODO: Accesses to this.intr here is racy
	} else if refcnt == 0 && this.intr != nil {
		close(this.intr)
		this.intr = nil
	}
}

func (this *valueImpl) resetFetchOnce() {
	this.fetchOnceLock.Lock()
	defer this.fetchOnceLock.Unlock()

	this.fetchOnce = &sync.Once{}
}

func (this *valueImpl) beginFetch() {
	this.fetchOnceLock.RLock()
	defer this.fetchOnceLock.RUnlock()

	this.fetchOnce.Do(func() { go this.fetch() })
}
