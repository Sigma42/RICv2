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

func maybe_register_serial(port_name string, r *Router) {
	fmt.Println("try:",port_name)
	if !SELECTORS.Match([]byte(port_name)) {
		return
	}

	fmt.Println("open:",port_name)

	port, err := serial.Open(port_name, &serial.Mode{BaudRate: BAUDRATE})
	if err != nil {
		log.Fatal(err)
	}
	port.SetReadTimeout(serial.NoTimeout)
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	fmt.Println("opened:",port_name)

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

		fmt.Println("handshake candidate:",buf)

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

	closeConnection := make(chan bool, 2) //Has to have buffer to prevent deadlock

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
			select {
			case <-closeConnection:
				return
			default:
				_, err = io.ReadFull(port, in_buf)
				if err != nil {
					closeConnection <- true
					return
				}

				fmt.Println("Recievd: ",in_buf,in_package)

				in_package, err = packagefromBytes(in_buf)
				if err != nil {
					log.Println("NOT HAPPEN !!!!")
					return //SHOULD NOT HAPPEN !!!!
				}

				if (in_package.flags() & 1) != 0 {
					continue //Ignore new Register (hopfully will handle restart of Serial-Client (where the connection stays open) )
				}

				r.route(in_package)
			}

		}

	}()

	go func() {
		defer wg.Done()

		for {
			select {
			case p, ok := <-channel:
				if !ok {
					closeConnection <- true
					return // The channel was closed, exit the goroutine
				}

				/*err := port.Drain()
				if err != nil {
					closeConnection <- true
					return
				}*/

				fmt.Println("Start sending:",p.toBytes())

				n, err := port.Write(p.toBytes())
				if err != nil {
					closeConnection <- true
					return
				}
				if n != PACKAGE_SIZE {
					log.Println("PACKAGE WAS NOT COMPLETLE SEND (send bytes WRONG SIZE). IGNORING!!")
					continue
				}

			case <-closeConnection: // This listens for a signal on the closeConnection channel
				return // Exit when true is sent on closeConnection
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
