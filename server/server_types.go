package main

import (
	"encoding/gob"
	"net"
)

type User struct {
	Connection net.Conn
	Logged     bool
	Username   string
	Password   string
	Encoder    *gob.Encoder
}
