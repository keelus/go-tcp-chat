package common

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

const TCP_CHAT_VERSION = "v0.1.1"

type Broadcast struct {
	Sender    string
	Content   string
	Date      time.Time
	Type      BroadcastType
	Printable bool
	Code      ResponseCode
	SentFrom  SendLocation
}

type BroadcastType int

const (
	TEXT     BroadcastType = 0
	ERROR    BroadcastType = 1
	MESSAGE  BroadcastType = 2
	ACTIVITY BroadcastType = 3
	VERSION  BroadcastType = 4
)

func (broadcast *Broadcast) RenderBroadcast() (string, tcell.Style) {
	style := tcell.StyleDefault
	rendered := fmt.Sprintf("%s ", RenderDate(broadcast.Date))

	switch broadcast.Type {
	case MESSAGE:
		rendered += fmt.Sprintf("<%s>%s", broadcast.Sender, broadcast.Content)
		style = style.Foreground(tcell.ColorWhite)
	case ERROR:
		rendered += fmt.Sprintf("[ERROR] %s", broadcast.Content)
		style = style.Foreground(tcell.ColorRed).Bold(true)
	case TEXT:
		rendered += broadcast.Content
		style = style.Foreground(tcell.Color102)
	case ACTIVITY:
		rendered += broadcast.Content
		style = style.Foreground(tcell.Color100)
	}
	return rendered, style
}

type ResponseCode int

const (
	C_ERROR ResponseCode = 0
	C_OK    ResponseCode = 1
)

type SendLocation string

const (
	L_CLIENT SendLocation = "C"
	L_SERVER SendLocation = "S"
)

func RenderDate(date time.Time) string {
	return date.Format("15:04:05")
}
