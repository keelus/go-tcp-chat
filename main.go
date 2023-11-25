package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const PORT = "6969"

var UserList []*User = make([]*User, 0)

type User struct {
	Username   string
	Password   string
	Connection net.Conn
}

type Message struct {
	Sender  string
	Content string
	Date    int
}

func sendUserMessage(message Message) {
	log.Printf("Trying to send a message...")
	for _, registeredUser := range UserList {
		if registeredUser.Connection != nil {
			registeredUser.Connection.Write([]byte(fmt.Sprintf("[%d]<%s> %s\n", message.Date, []byte(message.Sender), []byte(message.Content))))
		}
	}
}

func initialSetup(conn net.Conn) *User {
	connUser := User{Username: "____UNDEFINED____", Password: "", Connection: conn}

	reader := bufio.NewReader(conn)
	for {
		conn.Write([]byte("Choose an username [4-15 chars]: "))

		desiredUsername, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error on reading username: '%s'\n", err.Error())
			conn.Close()
			return nil
		}

		desiredUsername = strings.TrimSpace(desiredUsername)

		if !(len(desiredUsername) >= 4 && len(desiredUsername) <= 15) {
			conn.Write([]byte("Incorrect. Username must be 4 or longer and 15 or less characters length.\n"))
			continue
		}

		connUser.Username = desiredUsername
		break
	}
	for {
		conn.Write([]byte("Choose a password [>5 chars]: "))

		desiredPassword, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error on reading username: '%s'\n", err.Error())
			conn.Close()
			return nil
		}

		desiredPassword = strings.TrimSpace(desiredPassword)

		if len(desiredPassword) < 5 {
			conn.Write([]byte("Incorrect. Password must be 5 characters length or greater.\n"))
			continue
		}

		connUser.Password = desiredPassword
		break
	}

	return &connUser
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.Write([]byte("🎉 Connected to the IRC server\n"))

	reader := bufio.NewReader(conn)
	for {
		connUser := initialSetup(conn)
		if connUser == nil {
			return
		}

		UserList = append(UserList, connUser)

		conn.Write([]byte("Registration successfully completed.\n"))

		// Main user input loop
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Error reading message: '%s'\n", err.Error())
				conn.Close()
				return
			}

			msg = strings.TrimSpace(msg)

			msgParts := strings.Split(msg, " ")

			switch msgParts[0] {
			case "/msg":
				log.Printf("%s sent a message\n", connUser.Username)

				msgContent := strings.Replace(msg, "/msg", "", 1)
				msgContent = fmt.Sprintf("%s\n", msgContent)
				dateNow := int(time.Now().Unix())
				message := Message{Sender: connUser.Username, Content: msgContent, Date: dateNow}
				go sendUserMessage(message)

				break
			case "/quit":
				log.Printf("%s disconnected\n", connUser.Username)
				conn.Write([]byte(fmt.Sprintf("Goodbye %s!\n", connUser.Username)))

				for i, connectedUser := range UserList {
					if connectedUser.Username == connUser.Username {
						UserList[i].Connection = nil
					}
				}

				return
			default:
				log.Printf("%s sent an unknown command\n", connUser.Username)
			}
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf("192.168.0.70:%s", PORT))
	if err != nil {
		log.Fatalf("An error happened while listening to the port %s\n", PORT)
	}
	defer listener.Close()

	log.Printf("Listening to the port %s...\n", PORT)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("There was an error handling the connection.\n")
		} else {
			log.Printf("Client [%s] connected.", conn.RemoteAddr())
			go handleConnection(conn)
		}
	}
}
