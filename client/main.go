package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"fmt"

	"github.com/gorilla/websocket"
)

func readLoop(conn *websocket.Conn){
	for {
		_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("maaan gg:", err)
				return
			}
		log.Printf("message got: %s", message)
	}
}

func main(){
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var host string
	fmt.Print("Type host you want to connect to: ")
	_, err := fmt.Scanln(&host)
	if err != nil {
		log.Println(err)
	}

	u := url.URL{Scheme: "ws", Host: host, Path: "/"}
	log.Printf("connecting to: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	conn.WriteMessage(websocket.TextMessage, []byte("celkem skibodi"))

	go readLoop(conn)
	for {
		select{
		case <-interrupt:
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	    if err != nil {
	    	log.Println("Chyba při odesílání close zprávy:", err)
				os.Exit(1)
	    }
	    os.Exit(0)
		}
	}
}
