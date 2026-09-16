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
    tea "charm.land/bubbletea/v2"
    "charm.land/bubbles/v2/textarea"
    "charm.land/bubbles/v2/viewport"
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
        msg_chan <- "< " + string(message)
    }
}

func setupPong(conn *websocket.Conn){
    conn.SetReadDeadline(time.Now().Add(pongWait))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(pongWait))
        log.Println("PONG JE TU")
        return nil
    })
}

type Output struct {
    message string
    msg_type int
}

func evalCommand(command string, conn *websocket.Conn) tea.Cmd {
    command = strings.TrimSpace(command)

    if command == "/exit" {
        err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
        if err != nil {
            log.Println("Chyba při odesílání close zprávy:", err)
        }
        return tea.Quit
    }
    return nil
}

func sendMessage(msg_to_send Output, conn *websocket.Conn){
    if msg_to_send.msg_type == websocket.TextMessage {
        err := conn.WriteMessage(websocket.TextMessage, []byte(msg_to_send.message))
        if err != nil {
            log.Println("Je to v pici: ", err)
            os.Exit(1)
        } else {
            log.Println("MSG SENT")
        }
    } else if msg_to_send.msg_type == websocket.PingMessage {
        conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
        log.Println("PINGUJU VRO")
        if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
            log.Println("PING SE DOSRAL", err)
            return 
        }   
    } 
}

type model struct {
    input textarea.Model
    output viewport.Model
    lines []string
    width, height int
    conn *websocket.Conn
    readChan chan string
}

const inputHeight = 1

type logWriter struct {
    ch chan string
}

func (w logWriter) Write(p []byte) (int, error) {
    w.ch <- strings.TrimRight(string(p), "\n")
    return len(p), nil
}

func (m *model) appendLine(line string) {
    m.lines = append(m.lines, line)
    m.output.SetContent(strings.Join(m.lines, "\n"))
    m.output.GotoBottom()
}

func (m *model) resize(w, h int) {
    m.width, m.height = w, h
    m.input.SetWidth(w)
    m.input.SetHeight(inputHeight)
    m.output.SetWidth(w)
    m.output.SetHeight(max(h - inputHeight - 1, 1))
    m.output.SetContent(strings.Join(m.lines, "\n"))
    m.output.GotoBottom()
}

type IncomingMsg string
type PingMsg struct{}

func tickPing() tea.Cmd {
    return tea.Tick(pingPeriod, func(time.Time) tea.Msg {
        return PingMsg{}
    })
}

func initModel(conn *websocket.Conn, readChan chan string) model{
    out := viewport.New()

    in := textarea.New()
    in.ShowLineNumbers = false
    in.Prompt = "> "
    in.SetHeight(inputHeight)
    in.Focus()
    in.Placeholder = "type message :D"

    return model{
        input: in, 
        output: out,
        conn: conn, 
        readChan: readChan,
    }
}

func (m model) Init() tea.Cmd {
    return tea.Batch(
        textarea.Blink,
        waitForWebSocket(m.readChan),
        tickPing(),
    )
}

func waitForWebSocket(helper chan string) tea.Cmd {
    return func() tea.Msg {
        msg := <- helper
        return IncomingMsg(msg)
    }
}

func (m model) View() tea.View {
    sep := strings.Repeat("─", m.width)
    v := tea.NewView(m.output.View() + "\n" + sep + "\n" + m.input.View())
    v.AltScreen = true
    return v
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.resize(msg.Width, msg.Height)
        return m, nil

    case tea.KeyPressMsg:
        switch msg.String() {
        case "ctrl+c", "esc":
            return m, tea.Quit

        case "enter":
            val := strings.TrimSpace(m.input.Value())
            if val != "" {
                m.input.Reset()
                if strings.HasPrefix(val, "/") {
                    return m, evalCommand(val, m.conn)
                }
                outObj := Output{message: val, msg_type: websocket.TextMessage}
                sendMessage(outObj, m.conn)
            }
            return m, nil
        }

    case PingMsg:
        sendMessage(Output{message: "ping", msg_type: websocket.PingMessage}, m.conn)
        return m, tickPing()

    case IncomingMsg:
        m.appendLine(string(msg))

        return m, waitForWebSocket(m.readChan)
    }

    m.input, cmd = m.input.Update(msg)
    cmds = append(cmds, cmd)

    m.output, cmd = m.output.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}


func main() {
    reader := bufio.NewReader(os.Stdin)
    read_chan := make(chan string, 256)
    log.SetOutput(logWriter{ch: read_chan})

    var host string
    fmt.Print("Type host you want to connect to: ")
    host, err := reader.ReadString('\n')
    if err != nil {
        log.Println(err)
    }
    host = strings.TrimSpace(host)
    var scheme string
    if strings.HasPrefix(host, "localhost") {
        scheme = "ws"
    } else {
        scheme = "wss"
    }

    u := url.URL{Scheme: scheme, Host: host, Path: "/"}
    log.Printf("connecting to: %s", u.String())

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatalln(err)
    }
    defer conn.Close()

    setupPong(conn)
    go readLoop(conn, read_chan)

    p := tea.NewProgram(initModel(conn, read_chan))
    if _, err := p.Run(); err != nil {
        log.Fatalln(err)
    }
}    