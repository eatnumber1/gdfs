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
	Done()
	Forget()
}

type ValueImpl struct {
	waiters int64
	intr fusefs.Intr
	shutdown chan struct{}
	forget chan struct{}
	result chan fetcherResult

	isShutdown int32 // bool

	// Only access using atomics
	fetchOnce *sync.Once
	fetchOnceLock sync.RWMutex
	fetchFunc FetchFunc
}

type ValueRef struct {
	*ValueImpl
}

type fetcherResult struct {
	value interface{}
	err error
}

func (this *ValueImpl) Get(intr fusefs.Intr) (interface{}, error) {
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

func (this *ValueImpl) Done() {
	v := atomic.SwapInt32(&this.isShutdown, 1)
	if v == 0 {
		close(this.shutdown)
	}
}

func finalizeRef(ref *ValueRef) {
	ref.Done()
}

func NewValue(fetchFunc FetchFunc) Value {
	impl := &ValueImpl{
		waiters: 0,
		intr: make(fusefs.Intr),
		shutdown: make(chan struct{}),
		forget: make(chan struct{}, 1),
		result: make(chan fetcherResult),
		fetchFunc: fetchFunc,
		fetchOnce: &sync.Once{},
		isShutdown: 0,
	}
	ref := &ValueRef{ impl }
	runtime.SetFinalizer(ref, finalizeRef)
	return ref
}

func (this *ValueImpl) Forget() {
	this.forget <- struct{}{}
}

func (this *ValueImpl) fetch() {
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
		case <-this.forget:
			this.resetFetchOnce()
			return
		case this.result <- res:
		}
	}
}

func (this *ValueImpl) ref() {
	atomic.AddInt64(&this.waiters, int64(1))
}

func (this *ValueImpl) unref() {
	refcnt := atomic.AddInt64(&this.waiters, int64(-1))
	if refcnt == -1 {
		panic("unref() called on value with zero waiters")
	} else if refcnt == 0 && this.intr != nil {
		close(this.intr)
		this.intr = nil
	}
}

func (this *ValueImpl) resetFetchOnce() {
	this.fetchOnceLock.Lock()
	defer this.fetchOnceLock.Unlock()

	this.fetchOnce = &sync.Once{}
}

func (this *ValueImpl) beginFetch() {
	this.fetchOnceLock.RLock()
	defer this.fetchOnceLock.RUnlock()

	this.fetchOnce.Do(func() { go this.fetch() })
}
