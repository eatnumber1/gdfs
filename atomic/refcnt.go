package atomic

import (
	sysAtomic "sync/atomic"
)

type ReferenceCounter struct {
	refcnt uint64
}

func NewReferenceCounter() ReferenceCounter {
	return ReferenceCounter{
		refcnt: 1,
	}
}

func (this ReferenceCounter) ref() {
	sysAtomic.AddUint64(&this.refcnt, uint64(1))
}

func (this ReferenceCounter) unref() bool {
	return sysAtomic.AddUint64(&this.refcnt, ^uint64(0)) != 0
}
