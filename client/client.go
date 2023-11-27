package main

import (
	"encoding/gob"
	"fmt"
	"go-tcp-chat/common"
	"log"
	"net"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type Client struct {
	Connection      net.Conn
	Decoder         gob.Decoder
	BroadcastBuffer []common.Broadcast
	UI              ScreenUI
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

	client.UI = ScreenUI{Screen: screen, Prompt: ""}
	client.UI.Screen.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite))
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
		client.UI.Draw(client.BroadcastBuffer)
	}
}

func (client *Client) RunUI() {
	for {
		client.UI.Draw(client.BroadcastBuffer)
		switch ev := client.UI.Screen.PollEvent().(type) {
		case *tcell.EventResize:
			client.UI.Screen.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				client.UI.Screen.Fini()
				os.Exit(0)
			case tcell.KeyEnter:
				client.Connection.Write([]byte(fmt.Sprintf("%s\n", client.UI.Prompt)))
				log.Printf("[INFO] Message sent")
				client.UI.Prompt = ""
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(client.UI.Prompt) > 0 {
					client.UI.Prompt = client.UI.Prompt[0 : len(client.UI.Prompt)-1]
				}
			case tcell.KeyRune:
				client.UI.Prompt += string(ev.Rune())
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
					client.UI.Draw(client.BroadcastBuffer)
					os.Exit(-1)
				}
			}
		}
	}
}
