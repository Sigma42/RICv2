package main

import (
	"os"
	"os/signal"
	"syscall"
)

func start(r *Router) {
	go main_serial(r)
	go main_ws(r)
}

func cleanup(r *Router) {
	r.cleanup()
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	r := newRouter()

	start(r)

	<-c
	cleanup(r)
	os.Exit(0)
}
