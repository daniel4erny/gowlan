package main

import (
	"net/http"
	"os"
	"github.com/gorilla/websocket"
	"log"
)

type Hub struct {
	conns map[string]*websocket.Conn
	reg_chan chan *websocket.Conn
	del_chan chan *websocket.Conn
	write_chan chan []byte
}

func newHub() *Hub {
	return &Hub{
		conns: make(map[string]*websocket.Conn),
		reg_chan: make(chan *websocket.Conn),
		del_chan: make(chan *websocket.Conn),
		write_chan: make(chan []byte),
	}
}

func (h *Hub) Register(conn *websocket.Conn){
	h.reg_chan <- conn
}

func (h *Hub) Delete(conn *websocket.Conn){
	h.del_chan <- conn
}

func (h *Hub) Broadcast(message []byte) {
    h.write_chan <- message
}

func (h *Hub) Run() {
	for{
		select{
		case conn := <- h.reg_chan:
			h.conns[conn.RemoteAddr().String()] = conn
			log.Println("REG: " + conn.RemoteAddr().String())

		case conn := <- h.del_chan:
			_, ok := h.conns[conn.RemoteAddr().String()]

			if ok {
				delete(h.conns, conn.RemoteAddr().String())
				conn.Close()
			}

			log.Println("DEL: " + conn.RemoteAddr().String())

		case message := <- h.write_chan:
			for id, conn := range h.conns{
				err := conn.WriteMessage(websocket.TextMessage, message)
				log.Printf("sending %s to %s", message, id)
				if err != nil {
					log.Println(err)
					delete(h.conns, id)
					conn.Close()
				}
			} 

			log.Println("MSG: " + string(message))
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Hub) Serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("WebSocket Server is alive!"))
        return
    }
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	h.Register(conn)

	go func(){
		defer h.Delete(conn)

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				break
			}
			h.Broadcast(message)
		}
	}()
}

func main(){
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	hub := newHub()
	go hub.Run()

	http.HandleFunc("/", hub.Serve)

	log.Println("READY")

	http.ListenAndServe(":"+port, nil)
}