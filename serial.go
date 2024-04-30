package main

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.bug.st/serial"
)

const BAUDRATE = 57600 //115200 //9600
var SELECTORS = regexp.MustCompile(`(tty[\s\S]*?(ACM))|((ACM)[\s\S]*?tty)`)

//TODO REINIT when client send HANDSHAKE

func maybe_register_serial(port_name string, r *Router) {
	if !SELECTORS.Match([]byte(port_name)) {
		return
	}

	port, err := serial.Open(port_name, &serial.Mode{BaudRate: BAUDRATE})
	if err != nil {
		log.Fatal(err)
	}
	port.SetReadTimeout(serial.NoTimeout)
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	var address uint8
	var channel chan *Package
	var snooping bool
	err = fmt.Errorf("dummy error")

	buf := make_package_bytes_buffer()

	for err != nil {
		var hP *Package

		_, err = io.ReadFull(port, buf)
		if err != nil {
			return
		}

		hP, err = packagefromBytes(buf)
		if err != nil {
			continue //Skip message with wrong size
		}

		address, channel, snooping, err = hP.asHandshake(r)
	}
	defer r.notifyDisconnected(address)

	if snooping {
		r.registerSnooper(address)
		defer r.unregisterSnooper(address)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		//not recreate package
		in_buf := make_package_bytes_buffer()
		in_package, err := packagefromBytes(in_buf)
		if err != nil {
			log.Println("NOT HAPPEN !!!!")
			return //SHOULD NOT HAPPEN !!!!
		}

		for {
			_, err = io.ReadFull(port, in_buf)
			if err != nil {
				return
			}

			r.route(in_package)
		}

	}()
	go func() {
		defer wg.Done()

		for p := range channel {
			err := port.Drain()
			if err != nil {
				return
			}

			n, err := port.Write(p.toBytes())
			if err != nil {
				return
			}
			if n != PACKAGE_SIZE {
				log.Println("PACKAGE WAS NOT COMPLETLE SEND (send bytes WRONG SIZE). IGNORING!!")
				continue
			}
		}
	}()

	wg.Wait()
}

func main_serial(r *Router) {

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}

	for _, port := range ports {
		go maybe_register_serial(port, r)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add a path.
	err = watcher.AddWith("/dev")
	if err != nil {
		log.Fatal(err)
	}

	//Handle Updates
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) {
				go maybe_register_serial(event.Name, r)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}
