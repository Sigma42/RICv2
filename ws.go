package main

import (
	"embed"
	_ "embed"
	"log"
	"net/http"
	"sync"

	"github.com/fasthttp/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func ws(router *Router) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()

		//Handshake
		mt, message, err := c.ReadMessage()
		if err != nil || mt != websocket.BinaryMessage {
			return
		}
		hP, err := packagefromBytes(message)
		if err != nil {
			return
		}
		address, channel, snooping := hP.asHandshake(router)
		if snooping {
			router.registerSnooper(address)
			defer router.unregisterSnooper(address)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()

			for {
				mt, message, err := c.ReadMessage()
				if err != nil {
					break //Close on error
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

		}()
		go func() {
			defer wg.Done()

			for p := range channel {
				err = c.WriteMessage(websocket.BinaryMessage, p.toBytes())
				if err != nil {
					break
				}
			}
		}()

		wg.Wait()
	}

}

//go:embed index.html
var content embed.FS

func main_ws(r *Router) {
	http.HandleFunc("/", ws(r))
	http.Handle("/console", http.StripPrefix("/console", http.FileServer(http.FS(content))))

	http.ListenAndServe("0.0.0.0:8080", nil)
}
