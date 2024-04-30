package main

import (
	"fmt"
	"sync"
)

/*type Package struct {
	version  uint8
	src      uint8
	dst      uint8
	flags	 uint8 -> MSB[ disconnted (src) ,Unused,Unused,Unused,Unused,Unused,snoop,register  ]LSB
	data     [16]byte
}*/

const PACKAGE_SIZE = 24

type Package struct {
	_internal []byte
}

func make_package_bytes_buffer() []byte {
	buf := make([]byte, PACKAGE_SIZE)

	buf[0] = 1

	return buf
}

func packagefromBytes(bytes []byte) (*Package, error) {
	if bytes == nil || len(bytes) != PACKAGE_SIZE {
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

func (p *Package) flags() uint8 {
	return p._internal[3]
}

func (p *Package) data() []byte {
	return p._internal[4:]
}

func (p *Package) asHandshake(r *Router) (uint8, chan *Package, bool, error) {
	if p.version() != 1 {
		return 0, nil, false, fmt.Errorf("wrong Version Number")
	}

	if (p.flags() & 1) == 0 {
		return 0, nil, false, fmt.Errorf("not a register package")
	}

	src := p.src()
	wantsToSnoop := (p.flags() & 2) != 0

	r.sendToSnooper(p) //ALSO send succesfull handshake to Snooper
	return src, r.channels[src], wantsToSnoop, nil
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

	r.sendToSnooper(p)

	r.chanMutex.RUnlock()
}

func (r *Router) sendToSnooper(p *Package) {
	r.snooperChannels.Range(func(_ any, value any) bool {
		var channel chan *Package = value.(chan *Package)

		channel <- p

		return true
	})
}

func (r *Router) registerSnooper(idx uint8) {
	r.snooperChannels.Store(idx, r.channels[idx])
}

func (r *Router) unregisterSnooper(idx uint8) {
	r.snooperChannels.Delete(idx)
}

func (r *Router) notifyDisconnected(src uint8) {

	buf := make_package_bytes_buffer()
	buf[1] = src
	buf[3] = 128

	p, err := packagefromBytes(buf)
	if err != nil {
		fmt.Printf("This should not have happend!!")
	}

	r.sendToSnooper(p)
}

func (r *Router) cleanup() {
	r.chanMutex.Lock()

	for i := range 256 {
		close(r.channels[i])
	}

	//Dont Unlock (route method is not valid any more)
	//r.chanMutex.Unlock()
}
