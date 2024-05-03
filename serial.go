package main

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.bug.st/serial"
)

const BAUDRATE = 57600 //115200 //9600
var SELECTORS = regexp.MustCompile(`(tty[\s\S]*?(ACM))|((ACM)[\s\S]*?tty)`)
var SELECTORS_MAC = regexp.MustCompile(`cu\.usbmodem|tty.usbmodem`)
var SELECTORS_WIN = regexp.MustCompile(`COM`)

func maybe_register_serial(port_name string, r *Router) {
	fmt.Println("try:", port_name)
	if !SELECTORS.Match([]byte(port_name)) && !SELECTORS_MAC.Match([]byte(port_name)) && !SELECTORS_WIN.Match([]byte(port_name)) {
		return
	}

	fmt.Println("open:", port_name)

	port, err := serial.Open(port_name, &serial.Mode{BaudRate: BAUDRATE})
	if err != nil {
		log.Println("Error: ", err)
	}
	err = port.SetReadTimeout(time.Second * 20) //Set timeout for Handshake to 20 Sekonds per connection
	if err != nil {
		log.Println("Error: ", err)
	}
	defer port.Close()

	fmt.Println("opened:", port_name)

	var address uint8
	var channel chan *Package
	var snooping bool
	var has_dynamic_src bool
	err = fmt.Errorf("dummy error")

	var hP *Package
	buf := make_package_bytes_buffer()

	try := 0
	for err != nil {
		try = try + 1
		if try > 10 {
			fmt.Println("Handshake faild for: ", port_name)
			return
		}

		_, err = io.ReadFull(port, buf)
		if err != nil {
			return
		}

		fmt.Println("handshake candidate:", buf)

		hP, err = packagefromBytes(buf)
		if err != nil {
			continue //Skip message with wrong size
		}

		address, channel, snooping, has_dynamic_src, err = hP.asHandshake(r)
	}
	defer r.notifyDisconnected(address)

	err = port.SetReadTimeout(serial.NoTimeout) //Disable timeout
	if err != nil {
		log.Println("Error: ", err)
	}

	if has_dynamic_src {
		n, err := port.Write(hP.toBytes()) //HP sollte korrekte neue SRC haben (wegen asHandshake)
		if err != nil {
			return
		}
		if n != PACKAGE_SIZE {
			log.Println("PACKAGE WAS NOT COMPLETLE SEND (send bytes WRONG SIZE). IGNORING!!")
			return
		}
	}

	if snooping {
		r.registerSnooper(address)
		defer r.unregisterSnooper(address)
	}

	closeConnection := make(chan bool, 2) //Has to have buffer to prevent deadlock

	r.connectedClientsAdd(address)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-closeConnection:
				return
			default:
				in_buf := make_package_bytes_buffer()
				in_package, err := packagefromBytes(in_buf)
				if err != nil {
					log.Println("NOT HAPPEN !!!!")
					return //SHOULD NOT HAPPEN !!!!
				}

				_, err = io.ReadFull(port, in_buf)
				if err != nil {
					closeConnection <- true
					return
				}

				fmt.Println("Recievd: ", in_buf)

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

				fmt.Println("Start sending:", p.toBytes())

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
		log.Println(err)
	}

	for _, port := range ports {
		go maybe_register_serial(port, r)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
	}
	defer watcher.Close()

	// Add a path.
	err = watcher.AddWith("/dev")
	if err != nil {
		log.Println(err)
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
