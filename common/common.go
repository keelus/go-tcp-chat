package common

type Broadcast struct {
	Sender    string
	Content   string
	Date      int64
	Type      BroadcastType
	Printable bool
	Code      ResponseCode
	SentFrom  SendLocation
}

type BroadcastType int

const (
	TEXT    BroadcastType = 0
	ERROR   BroadcastType = 1
	MESSAGE BroadcastType = 2
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
