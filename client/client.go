package main

import (
	"encoding/gob"
	"fmt"
	"go-irc/common"
	"net"
	"os"

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
var logMsg = ""

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
	rendered := fmt.Sprintf("[%s][%d]", broadcast.SentFrom, broadcast.Date)

	switch broadcast.Type {
	case common.MESSAGE:
		rendered += fmt.Sprintf("<%s>%s", broadcast.Sender, broadcast.Content)
		style.Foreground(tcell.ColorWhite)
	case common.ERROR:
		rendered += fmt.Sprintf("[ERR]%s", broadcast.Content)
		style.Foreground(tcell.ColorRed)
	case common.TEXT:
		rendered += broadcast.Content
		style.Foreground(tcell.ColorBlueViolet)
	}
	return rendered, style
}

func drawChat(s tcell.Screen) {
	emitStr(s, 0, 0, tcell.StyleDefault, fmt.Sprintf("Length chat history: %d", len(chatHistory)))
	for i, broadcast := range chatHistory {
		renderedBroadcast, tcellStyle := renderBroadcast(broadcast)
		emitStr(s, 0, i+1, tcellStyle, renderedBroadcast)
	}
}

func drawPrompt(s tcell.Screen, msg string) {
	_, h := s.Size()
	emitStr(s, 0, h-1, tcell.StyleDefault, msg)
}

func drawLog(s tcell.Screen, logMsg string) {
	emitStr(s, 0, 30, tcell.StyleDefault, logMsg)
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
	// buffer := make([]byte, 512)
	decoder := gob.NewDecoder(conn)
	for {
		var receivedBroadcast common.Broadcast
		err := decoder.Decode(&receivedBroadcast)
		if err != nil {
			logMsg = fmt.Sprintf("Error decoding and receiving: %s", err.Error())
			return
		}
		logMsg = fmt.Sprintf("Received broadcast from server: %s", receivedBroadcast.Content)

		if receivedBroadcast.Printable {
			chatHistory = append(chatHistory, receivedBroadcast)
		}

		drawChat(s)
		drawLog(s, logMsg)
		drawSeparator(s)
		drawPrompt(s, prompt)
		s.Show()
		s.Clear()
	}
}

func main() {
	conn, err := net.Dial("tcp", "192.168.0.70:6969")
	if err != nil {
		fmt.Println("error conneting")
		os.Exit(1)
	}
	defer conn.Close()

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

	go listenToServer(conn, s)

	for {
		drawChat(s)
		drawLog(s, logMsg)
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
				logMsg = "sent!"
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
