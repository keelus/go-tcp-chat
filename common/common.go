package common

import (
	"time"
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
