package main

import (
	"flag"
	"fmt"
	"go-tcp-chat/common"
	"log"
	"os"

	_ "github.com/gdamore/tcell/v2/encoding"
)

const CLIENT_VERSION = common.TCP_CHAT_VERSION

const (
	LOGIN      int = 0
	MESSAGE        = 1
	DISCONNECT     = 2
)

const PROMPT_HEIGHT = 1

func initializeLogging() *os.File {
	logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("[ERROR] Error opening log file:", err)
	}
	os.Truncate("client.log", 0)
	log.SetOutput(logFile)
	return logFile
}

func main() {
	argIp := flag.String("ip", "127.0.0.1", "The server's IP")
	argPort := flag.String("port", "6969", "The port")
	flag.Parse()

	logFile := initializeLogging()
	defer logFile.Close()

	client := Client{}
	if err := client.Connect(*argIp, *argPort); err != nil {
		log.Printf("[ERROR] Connection to %s:%s could not be stablished.\n", *argIp, *argPort)
		os.Exit(1)
	}
	defer client.Connection.Close()

	if err := client.InitializeUI(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	go client.Run()
	client.RunUI()
}
