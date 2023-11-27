package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"go-irc/common"
	"log"
	"math"
	"net"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/encoding"

	"github.com/mattn/go-runewidth"
)

const (
	LOGIN      int = 0
	MESSAGE        = 1
	DISCONNECT     = 2
)

const PROMPT_HEIGHT = 1

var chatHistory []common.Broadcast = make([]common.Broadcast, 0)

var prompt = ""

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		s.SetContent(x, y, c, comb, style)
		x += w
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

func drawChat(s tcell.Screen) {
	_, maxEntries := s.Size()
	maxEntries -= 2

	entriesFrom := int(math.Max(0, float64(len(chatHistory)-maxEntries)))
	entriesTo := len(chatHistory)

	entriesDrawn := 0
	for i := entriesFrom; i < entriesTo; i++ {
		if entriesDrawn == maxEntries {
			break
		}
		renderedBroadcast, tcellStyle := renderBroadcast(chatHistory[i])
		emitStr(s, 0, entriesDrawn, tcellStyle, renderedBroadcast)
		entriesDrawn += 1
	}
}

func drawPrompt(s tcell.Screen, msg string) {
	_, h := s.Size()
	emitStr(s, 0, h-1, tcell.StyleDefault, msg)
}

func drawSeparator(s tcell.Screen) {
	w, h := s.Size()
	separator := ""
	for i := 0; i < w; i++ {
		separator = separator + "="
	}
	emitStr(s, 0, h-2, tcell.StyleDefault, separator)
}

func listenToServer(conn net.Conn, s tcell.Screen) {
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

		if receivedBroadcast.Printable {
			msgLines := strings.Split(strings.TrimSpace(receivedBroadcast.Content), "\n")
			for _, line := range msgLines {
				tempBroadcast := receivedBroadcast
				tempBroadcast.Content = line
				log.Printf("b cnt: '%s'", tempBroadcast.Content)
				chatHistory = append(chatHistory, tempBroadcast)
			}
		}

		drawChat(s)
		drawSeparator(s)
		drawPrompt(s, prompt)
		s.Show()
		s.Clear()
	}
	// drawChat(s)
	// drawSeparator(s)
	// drawPrompt(s, prompt)
	// s.Show()
	// s.Clear()
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

	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	defStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)

	log.Printf("[INFO] Listening to server(%s:%s) broadcasts...", *argIp, *argPort)
	go listenToServer(conn, s)

	for {
		drawChat(s)
		drawSeparator(s)
		drawPrompt(s, prompt)
		s.Show()
		s.Clear()

		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				s.Fini()
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
