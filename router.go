package main

import (
	"fmt"
	"sync"
)

/*type Package struct {
	version  uint8
	src      uint8
	dst      uint8
	reserved uint8
	data     [16]byte
}*/

type Package struct {
	_internal []byte
}

func packagefromBytes(bytes []byte) (*Package, error) {
	if bytes == nil || len(bytes) != 24 {
		return nil, fmt.Errorf("wrong size")
	}

	return &Package{_internal: bytes}, nil
}

func (p *Package) toBytes() []byte {
	return p._internal[:]
}

func (p *Package) version() uint8 {
	return p._internal[0]
}

func (p *Package) src() uint8 {
	return p._internal[1]
}

func (p *Package) dst() uint8 {
	return p._internal[2]
}

func (p *Package) data() []byte {
	return p._internal[4:]
}

func (p *Package) asHandshake(r *Router) (uint8, chan *Package, bool) {
	var src uint8 = p._internal[1]
	var wantsToSnoop = p._internal[2] > 0

	return src, r.channels[src], wantsToSnoop
}

type Router struct {
	channels []chan *Package

	snooperChannels sync.Map
	chanMutex       sync.RWMutex
}

func newRouter() *Router {
	r := Router{
		channels: make([]chan *Package, 256),
	}

	for i := range 256 {
		r.channels[i] = make(chan *Package, 256)
	}

	return &r
}

func (r *Router) route(p *Package) {
	r.chanMutex.RLock()

	r.channels[p.dst()] <- p

	r.snooperChannels.Range(func(_ any, value any) bool {
		var channel chan *Package = value.(chan *Package)

		channel <- p

		return true
	})

	r.chanMutex.RUnlock()
}

func (r *Router) registerSnooper(idx uint8) {
	r.snooperChannels.Store(idx, r.channels[idx])
}

func (r *Router) unregisterSnooper(idx uint8) {
	r.snooperChannels.Delete(idx)
}

func (r *Router) cleanup() {
	r.chanMutex.Lock()

	for i := range 256 {
		close(r.channels[i])
	}

	//Dont Unlock (route method is not valid any more)
	//r.chanMutex.Unlock()
}
