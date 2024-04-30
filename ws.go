package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/fasthttp/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func ws(router *Router) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()

		var address uint8
		var channel chan *Package
		var snooping bool
		err = fmt.Errorf("dummy error")

		for err != nil { //Wait until register message
			var mt int
			var message []byte
			var hP *Package

			//Handshake
			mt, message, err = c.ReadMessage()
			if err != nil {
				return //Close on error
			}
			if mt != websocket.BinaryMessage {
				continue //Skip message with wrong type
			}

			hP, err = packagefromBytes(message)
			if err != nil {
				continue //Skip message with wrong size
			}

			address, channel, snooping, err = hP.asHandshake(router)
		}

		closeConnection := make(chan bool)
		c.SetCloseHandler(func(code int, text string) error {
			closeConnection <- true
			router.notifyDisconnected(address)
			return nil
		})

		if snooping {
			router.registerSnooper(address)
			defer router.unregisterSnooper(address)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-closeConnection:
					return
				default:
					mt, message, err := c.ReadMessage()
					if err != nil {
						return //Close on error
					}
					if mt != websocket.BinaryMessage {
						continue //Skip message with wrong type
					}

					p, err := packagefromBytes(message)
					if err != nil {
						continue //Skip message with wrong size
					}

					router.route(p)
				}

			}

		}()

		go func() {
			defer wg.Done()

			for {
				select {
				case p, ok := <-channel:
					if !ok {
						return // The channel was closed, exit the goroutine
					}
					err = c.WriteMessage(websocket.BinaryMessage, p.toBytes())
					if err != nil {
						return // Exit on error
					}
				case <-closeConnection: // This listens for a signal on the closeConnection channel
					return // Exit when true is sent on closeConnection
				}
			}
		}()

		wg.Wait()
	}

}

//go:embed console/*
var content embed.FS

func main_ws(r *Router) {
	http.HandleFunc("/", ws(r))

	http.Handle("/console/*", http.FileServer(http.FS(content)))

	http.ListenAndServe("0.0.0.0:8080", nil)
}
