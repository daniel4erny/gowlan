package main

import (
    "log"
    "net/url"
    "os"
    "bufio"
    "fmt"
    "strings"
    "time"

    "github.com/gorilla/websocket"
)

const (
    pingPeriod = 10 * time.Second
    pongWait = 15 * time.Second
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

func pingLoop(ping_err_chan chan bool, conn *websocket.Conn){
    pingTicker := time.NewTicker(pingPeriod)
    defer pingTicker.Stop()
    
    conn.SetReadDeadline(time.Now().Add(pongWait))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(pongWait))
        log.Println("PONG JE TU")
        return nil
    })

    for range pingTicker.C {
        conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
        log.Println("PINGUJU VRO")
        if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
            log.Println("PING SE DOSRAL", err)
            ping_err_chan <- true
            return 
        }
    }
}

func evalCommand(command string, conn *websocket.Conn){
    command = strings.TrimSpace(command)

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

    ping_err_chan := make(chan bool)
    go pingLoop(ping_err_chan, conn)

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
        case _ = <- ping_err_chan:
            evalCommand("/exit", conn)
        }
    }
}