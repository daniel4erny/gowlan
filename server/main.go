package main

import (
	"fmt"
	"net/http"
	"os"
	"github.com/gorilla/websocket"
	"log"
)

type Hub = struct {
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


var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main(){
	fmt.Println("Hello world!")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handleConnect)

	http.ListenAndServe(":"+port, nil)
}

func handleConnect(w http.ResponseWriter, r *http.Request){
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil{
		log.Println(err)
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil{
			log.Println(err)
			return
		}

		if messageType == websocket.TextMessage {
			log.Println(string(p))
		} else if messageType == websocket.CloseMessage {
			log.Println("end connection")
			break
		}
	}
}
