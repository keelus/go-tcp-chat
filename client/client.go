package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"go-tcp-chat/common"
	"log"
	"math"
	"net"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/encoding"

	"github.com/mattn/go-runewidth"
)

const CLIENT_VERSION = common.TCP_CHAT_VERSION

const (
	LOGIN      int = 0
	MESSAGE        = 1
	DISCONNECT     = 2
)

const PROMPT_HEIGHT = 1

var chatHistory []common.Broadcast = make([]common.Broadcast, 0)

var screen tcell.Screen

var prompt = ""

func emitStr(x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		screen.SetContent(x, y, c, comb, style)
		x += w
	}
}

func processBroadcast(conn net.Conn, broadcast common.Broadcast) {
	if broadcast.Printable {
		msgLines := strings.Split(strings.TrimSpace(broadcast.Content), "\n")
		for _, line := range msgLines {
			splittedBroadcastLine := broadcast
			splittedBroadcastLine.Content = line
			chatHistory = append(chatHistory, splittedBroadcastLine)
		}
	} else {
		if broadcast.Type == common.VERSION {
			switch broadcast.Type {
			case common.VERSION:
				server_version := strings.TrimSpace(broadcast.Content)
				if server_version != CLIENT_VERSION {
					log.Printf("NO SAME VERSION ERR")
					localBroadcast := common.Broadcast{
						Sender:    "__CLIENT__",
						Content:   fmt.Sprintf("You need to use the same version as the server. \n   Server version:%s\n   Your (client) version: %s", server_version, CLIENT_VERSION),
						Type:      common.ERROR,
						Printable: true,
						Code:      common.C_ERROR,
					}
					processBroadcast(conn, localBroadcast)
					conn.Close()
					drawScreen()
					os.Exit(-1)
				}
			}
		}
	}

}

func renderBroadcast(broadcast common.Broadcast) (string, tcell.Style) {
	style := tcell.StyleDefault
	rendered := fmt.Sprintf("%s ", common.RenderDate(broadcast.Date))

	switch broadcast.Type {
	case common.MESSAGE:
		rendered += fmt.Sprintf("<%s>%s", broadcast.Sender, broadcast.Content)
		style = style.Foreground(tcell.ColorWhite)
	case common.ERROR:
		rendered += fmt.Sprintf("[ERROR] %s", broadcast.Content)
		style = style.Foreground(tcell.ColorRed).Bold(true)
	case common.TEXT:
		rendered += broadcast.Content
		style = style.Foreground(tcell.Color102)
	case common.ACTIVITY:
		rendered += broadcast.Content
		style = style.Foreground(tcell.Color100)
	}
	return rendered, style
}

func drawChat() {
	_, maxEntries := screen.Size()
	maxEntries -= 2

	entriesFrom := int(math.Max(0, float64(len(chatHistory)-maxEntries)))
	entriesTo := len(chatHistory)

	entriesDrawn := 0
	for i := entriesFrom; i < entriesTo; i++ {
		if entriesDrawn == maxEntries {
			break
		}
		renderedBroadcast, tcellStyle := renderBroadcast(chatHistory[i])
		emitStr(0, entriesDrawn, tcellStyle, renderedBroadcast)
		entriesDrawn += 1
	}
}

func drawPrompt(msg string) {
	_, h := screen.Size()
	emitStr(0, h-1, tcell.StyleDefault, msg)
}

func drawSeparator() {
	w, h := screen.Size()
	separator := ""
	for i := 0; i < w; i++ {
		separator = separator + "="
	}
	emitStr(0, h-2, tcell.StyleDefault, separator)
}

func drawScreen() {
	drawChat()
	drawSeparator()
	drawPrompt(prompt)
	screen.Show()
	screen.Clear()
}

func listenToServer(conn net.Conn) {
	decoder := gob.NewDecoder(conn)

	for {
		var receivedBroadcast common.Broadcast
		err := decoder.Decode(&receivedBroadcast)
		if err != nil {
			if netErr, ok := err.(*net.OpError); ok && netErr.Temporary() {
				log.Printf("[ERROR] Temporary error: %s\n", err.Error())
			} else {
				log.Printf("[INFO] Client disconnected\n")
				os.Exit(0)
			}

			log.Printf("[ERROR] Error decoding and receiving: %s", err.Error())
			return
		}
		log.Printf("[INFO] Received broadcast from server: %s", receivedBroadcast.Content)

		processBroadcast(conn, receivedBroadcast)

		drawScreen()
	}
}

func main() {
	argIp := flag.String("ip", "127.0.0.1", "The local IP")
	argPort := flag.String("port", "6969", "The port")
	flag.Parse()

	logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("[ERROR] Error opening log file:", err)
	}
	defer logFile.Close()
	os.Truncate("client.log", 0)
	log.SetOutput(logFile)

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", *argIp, *argPort))
	if err != nil {
		log.Printf("[ERROR] Connection to %s:%s could not be stablished.\n", *argIp, *argPort)
		os.Exit(1)
	}
	defer conn.Close()

	log.Printf("[INFO] Connection to %s:%s stablished.\n", *argIp, *argPort)
	encoding.Register()

	screen, err = tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err = screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	defStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	screen.SetStyle(defStyle)

	processBroadcast(conn, common.Broadcast{
		Sender:    "__CLIENT__",
		Content:   fmt.Sprintf("Running client on version %s", CLIENT_VERSION),
		Type:      common.TEXT,
		Printable: true,
		Code:      common.C_OK,
	})
	log.Printf("[INFO] Listening to server(%s:%s) broadcasts...", *argIp, *argPort)
	go listenToServer(conn)

	for {
		drawScreen()
		log.Printf("Drawn")

		switch ev := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				screen.Fini()
				os.Exit(0)
			case tcell.KeyEnter:
				conn.Write([]byte(fmt.Sprintf("%s\n", prompt)))
				log.Printf("[INFO] Message sent")
				prompt = ""
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(prompt) > 0 {
					prompt = prompt[0 : len(prompt)-1]
				}
			case tcell.KeyRune:
				prompt += string(ev.Rune())
			}
		}
	}
}
