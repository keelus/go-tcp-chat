package main

import (
	"encoding/gob"
	"fmt"
	"go-tcp-chat/common"
	"log"
	"math"
	"net"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type Client struct {
	Connection      net.Conn
	Decoder         gob.Decoder
	Screen          tcell.Screen
	Prompt          string
	BroadcastBuffer []common.Broadcast
}

func (client *Client) Connect(ip string, port string) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		return err
	}

	client.Connection = conn
	client.Decoder = *gob.NewDecoder(client.Connection)
	log.Printf("[INFO] Connection to %s:%s stablished.\n", ip, port)
	return nil
}

func (client *Client) InitializeUI() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = screen.Init(); err != nil {
		return err
	}

	client.Screen = screen
	screen.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite))
	return nil
}

func (client *Client) Run() {
	ip := "temp"
	port := "temp"
	log.Printf("[INFO] Listening to server(%s:%s) broadcasts...", ip, port)
	client.processBroadcast(common.Broadcast{
		Sender:    "__CLIENT__",
		Content:   fmt.Sprintf("Running client on version %s", CLIENT_VERSION),
		Type:      common.TEXT,
		Printable: true,
		Code:      common.C_OK,
	})

	for {
		var receivedBroadcast common.Broadcast
		err := client.Decoder.Decode(&receivedBroadcast)
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

		client.processBroadcast(receivedBroadcast)
		client.Draw()
	}
}

func (client *Client) RunUI() {
	for {
		client.Draw()
		switch ev := client.Screen.PollEvent().(type) {
		case *tcell.EventResize:
			client.Screen.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				client.Screen.Fini()
				os.Exit(0)
			case tcell.KeyEnter:
				client.Connection.Write([]byte(fmt.Sprintf("%s\n", client.Prompt)))
				log.Printf("[INFO] Message sent")
				client.Prompt = ""
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(client.Prompt) > 0 {
					client.Prompt = client.Prompt[0 : len(client.Prompt)-1]
				}
			case tcell.KeyRune:
				client.Prompt += string(ev.Rune())
			}
		}
	}
}

func (client *Client) processBroadcast(broadcast common.Broadcast) {
	if broadcast.Printable {
		msgLines := strings.Split(strings.TrimSpace(broadcast.Content), "\n")
		for _, line := range msgLines {
			splittedBroadcastLine := broadcast
			splittedBroadcastLine.Content = line
			client.BroadcastBuffer = append(client.BroadcastBuffer, splittedBroadcastLine)
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
					client.processBroadcast(localBroadcast)
					client.Connection.Close()
					client.Draw()
					os.Exit(-1)
				}
			}
		}
	}
}

func (client *Client) Draw() {
	client.drawChat()
	client.drawSeparator()
	client.drawPrompt(client.Prompt)
	client.Screen.Show()
	client.Screen.Clear()
}

func (client *Client) EmitStr(x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		client.Screen.SetContent(x, y, c, comb, style)
		x += w
	}
}

func (client *Client) RenderBroadcast(broadcast common.Broadcast) (string, tcell.Style) {
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

func (client *Client) drawChat() {
	_, maxEntries := client.Screen.Size()
	maxEntries -= 2

	entriesFrom := int(math.Max(0, float64(len(client.BroadcastBuffer)-maxEntries)))
	entriesTo := len(client.BroadcastBuffer)

	entriesDrawn := 0
	for i := entriesFrom; i < entriesTo; i++ {
		if entriesDrawn == maxEntries {
			break
		}
		renderedBroadcast, tcellStyle := client.RenderBroadcast(client.BroadcastBuffer[i])
		client.EmitStr(0, entriesDrawn, tcellStyle, renderedBroadcast)
		entriesDrawn += 1
	}
}

func (client *Client) drawPrompt(msg string) {
	_, h := client.Screen.Size()
	client.EmitStr(0, h-1, tcell.StyleDefault, msg)
}

func (client *Client) drawSeparator() {
	w, h := client.Screen.Size()
	separator := ""
	for i := 0; i < w; i++ {
		separator = separator + "="
	}
	client.EmitStr(0, h-2, tcell.StyleDefault, separator)
}
