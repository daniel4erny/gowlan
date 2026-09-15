package main

import (
    "log"
    "net/url"
    "os"
    "bufio"
    "fmt"
    "strings"

    "github.com/gorilla/websocket"
)



func readLoop(conn *websocket.Conn, msg_chan chan string) {
    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("maaan gg:", err)
            return
        }
        msg_chan <- string(message)
    }
}

func writeLoop(write_chan chan string, reader bufio.Reader){
    for {
        msg, err := reader.ReadString('\n')
        if err != nil {
            log.Println(err)
        }

        write_chan <- msg
    }
}

func evalCommand(command string, conn *websocket.Conn){
    if command == "/exit" {
        err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
        if err != nil {
            log.Println("Chyba při odesílání close zprávy:", err)
            os.Exit(1)
        }
        os.Exit(0)
    }
}

func sendMessage(msg_to_send string, conn *websocket.Conn){
    err := conn.WriteMessage(websocket.TextMessage, []byte(msg_to_send))
    if err != nil {
        log.Println("Je to v pici: ", err)
        os.Exit(1)
    } else {
        log.Println("odeslana zprava")
    }
}

func main() {
    reader := bufio.NewReader(os.Stdin)
    var host string
    fmt.Print("Type host you want to connect to: ")
    host, err := reader.ReadString('\n')
    if err != nil {
        log.Println(err)
    }
    host = strings.TrimSpace(host)

    u := url.URL{Scheme: "ws", Host: host, Path: "/"}
    log.Printf("connecting to: %s", u.String())

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatalln(err)
    }
    defer conn.Close()

    read_chan := make(chan string)
    go readLoop(conn, read_chan)

    write_chan := make(chan string)
    go writeLoop(write_chan, *reader)


    for {
        select {
        case message := <- read_chan:
            log.Printf("Got a message %s", message)
        case msg_to_send := <- write_chan:
            if strings.HasPrefix(msg_to_send ,"/") {
                evalCommand(msg_to_send, conn)
            } else {
                sendMessage(msg_to_send, conn)
            }
        }
    }
}