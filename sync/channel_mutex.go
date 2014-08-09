package sync

import (
	"runtime"
	"log"
)

type ChannelMutex interface {
	Done()
	Lock() chan struct{}
	Unlock() chan struct{}
}

type channelMutexImpl struct {
	lockc chan struct{}
	unlockc chan struct{}
	shutdown chan struct{}
}

func NewChannelMutex() ChannelMutex {
	priv := channelMutexImpl{
		lockc: make(chan struct{}),
		unlockc: make(chan struct{}),
		shutdown: make(chan struct{}),
	}
	mutex := ChannelMutex(&priv)
	runtime.SetFinalizer(mutex, finalizer)
	go priv.worker()
	return mutex
}

func finalizer(mutex ChannelMutex) {
	log.Printf("ChannelMutex.finalizer()")
	mutex.Done()
}

func (this channelMutexImpl) worker() {
	for done := false; done != true; {
		// Offer the lock
		select {
		case this.lockc <- struct{}{}:
		case <-this.shutdown:
			panic("Shutting down a held ChannelMutex")
		}

		// Wait for release
		select {
		case <-this.unlockc:
		case <-this.shutdown:
			done = true
		}
	}

	close(this.lockc)
	close(this.unlockc)
}

func (this channelMutexImpl) Done() {
	close(this.shutdown)
	this.shutdown = nil
}

func (this channelMutexImpl) Lock() chan struct{} {
	return this.lockc
}

func (this channelMutexImpl) Unlock() chan struct{} {
	return this.unlockc
}
