package util

import (
	"sync/atomic"
)

type ReferenceCount struct {
	refcnt uint64
}

func NewReferenceCount() *ReferenceCount {
	return &ReferenceCount{ 1 }
}

func (this *ReferenceCount) Ref() {
	atomic.AddUint64(&this.refcnt, uint64(1))
}

func (this *ReferenceCount) Unref() bool {
	return atomic.AddUint64(&this.refcnt, ^uint64(0)) == 0
}
