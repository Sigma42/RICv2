package main

import (
	"log"
	"regexp"

	"github.com/fsnotify/fsnotify"
	"go.bug.st/serial"
)

const BAUDRATE = 57600 //115200 //9600
var SELECTORS = regexp.MustCompile(`(tty[\s\S]*?(ACM))|((ACM)[\s\S]*?tty)`)

func maybe_register_serial(port_name string, r *Router) {
	if !SELECTORS.Match([]byte(port_name)) {
		return
	}

	_, err := serial.Open(port_name, &serial.Mode{BaudRate: BAUDRATE})
	if err != nil {
		log.Fatal(err)
	}
}

func handle_serial(port serial.Port, r *Router) {

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

}
