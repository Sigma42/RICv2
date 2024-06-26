package main

import (
	"fmt"
	"sync"
)

/*type Package struct {
	version  uint8
	src      uint8
	dst      uint8
	flags	 uint8 -> MSB[ disconnted (src) ,Unused,Unused,Unused,request-dynamic-address,request-list,snoop,register  ]LSB
	data     [20]byte
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
	return p._internal
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

func (p *Package) asHandshake(r *Router) (uint8, chan *Package, bool, bool, error) {
	if p.version() != 1 {
		return 0, nil, false, false, fmt.Errorf("wrong Version Number")
	}

	if (p.flags() & 1) == 0 {
		return 0, nil, false, false, fmt.Errorf("not a register package")
	}

	src := p.src()

	has_dynamic_src := (p.flags() & 8) != 0
	if has_dynamic_src {
		for i := 255; i >= 0; i-- {
			_, found := r.connectedClients.Load(uint8(i))
			if !found {
				src = uint8(i)

				p._internal[3] = p._internal[3] &^ 8 //Unset flag
				p._internal[1] = src                 //Correct in package for snoopers
				break
			}
		}
	}

	wantsToSnoop := (p.flags() & 2) != 0

	r.chanMutex.RLock()
	r.sendToSnooper(p, false, 0) //ALSO send succesfull handshake to Snooper
	r.chanMutex.RUnlock()

	if has_dynamic_src {
		p._internal[3] = p._internal[3] | 8 //Set the flag again (because we want to send to the original client)
	}

	return src, r.channels[src], wantsToSnoop, has_dynamic_src, nil
}

type Router struct {
	channels []chan *Package

	connectedClients sync.Map
	snooperChannels  sync.Map
	chanMutex        sync.RWMutex
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

	if (p.flags() & 4) != 0 { // Request list

		list := r.connectedClientsGet()

		for i := 0; i < len(list); i += 20 {
			buf := make_package_bytes_buffer()
			buf[0] = p.version()
			buf[1] = p.src()
			buf[2] = p.src()
			buf[3] = p.flags()

			j := 0
			for ; i+j < len(list) && j < 20; j += 1 {
				buf[j+4] = list[i+j]
			}
			for ; j < 20; j += 1 {
				buf[j+4] = buf[4]
			}

			newP, err := packagefromBytes(buf)
			if err != nil {
				fmt.Println("SHOULD NOT HAPPEN!!!")
			}

			r.channels[p.src()] <- newP
		}

	} else { // Normal Routing
		r.channels[p.dst()] <- p

		r.sendToSnooper(p, true, p.dst())
	}

	r.chanMutex.RUnlock()
}

func (r *Router) connectedClientsAdd(i uint8) {
	r.connectedClients.Store(i, i)
}

func (r *Router) connectedClientsRemove(i uint8) {
	r.connectedClients.Delete(i)
}

func (r *Router) connectedClientsGet() []uint8 {
	list := make([]uint8, 0, 24)

	r.connectedClients.Range(func(key any, value any) bool {
		list = append(list, key.(uint8))

		return true
	})

	return list
}

func (r *Router) sendToSnooper(p *Package, shouldIgnore bool, ignoreKey uint8) {
	buf := make_package_bytes_buffer()

	copy(buf, p.toBytes()) //make copy because race conditions

	p, err := packagefromBytes(buf)
	if err != nil {
		fmt.Println("Should not happen!!")
	}

	r.snooperChannels.Range(func(key any, value any) bool {
		if shouldIgnore && key.(uint8) == ignoreKey {
			return true
		}

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

	r.connectedClientsRemove(src)

	buf := make_package_bytes_buffer()
	buf[1] = src
	buf[3] = 128

	p, err := packagefromBytes(buf)
	if err != nil {
		fmt.Printf("This should not have happend!!")
	}

	r.sendToSnooper(p, false, 0)
}

func (r *Router) cleanup() {
	r.chanMutex.Lock()

	for i := range 256 {
		close(r.channels[i])
	}

	//Dont Unlock (route method is not valid any more)
	//r.chanMutex.Unlock()
}
