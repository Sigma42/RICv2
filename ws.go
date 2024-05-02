package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/fasthttp/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
} // use default options

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
		var has_dynamic_src bool
		err = fmt.Errorf("dummy error")
		var hP *Package

		for err != nil { //Wait until register message
			var mt int
			var message []byte

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

			address, channel, snooping, has_dynamic_src, err = hP.asHandshake(router)
		}

		if has_dynamic_src {
			err = c.WriteMessage(websocket.BinaryMessage, hP.toBytes()) //HP sollte korrekte neue SRC haben (wegen asHandshake)
			if err != nil {
				return // Exit on error
			}
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

		router.connectedClientsAdd(address)
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

//go:embed web/*
var content embed.FS

func fileServer(root http.FileSystem, defaultFile string) http.Handler {
	fs := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := root.Open(r.URL.Path)
		if err != nil { // File does not exist
			http.ServeFile(w, r, defaultFile)
			return
		}
		// Serve the actual file
		fs.ServeHTTP(w, r)
	})
}

func main_ws(r *Router) {
	http.HandleFunc("/ws", ws(r))

	staticFS, err := fs.Sub(content, "web")
	if err != nil {
		log.Fatalln("Should not happen!!!!")
	}

	http.Handle("/*", fileServer(http.FS(staticFS), "web/app.html"))

	http.ListenAndServe("0.0.0.0:8080", nil)
}
