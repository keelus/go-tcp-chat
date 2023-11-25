package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"go-irc/common"
	"log"
	"net"
	"strings"
	"time"
)

const PORT = "6969"

var UserList []*User = make([]*User, 0)

func sendBroadcast(user User, broadcast common.Broadcast) {

	err := user.Encoder.Encode(broadcast)
	if err != nil {
		log.Printf("Error sending the broadcast to '%s'\n", user.Connection.RemoteAddr())
	} else {
		log.Printf("Broadcast sent to '%s'\n", user.Connection.RemoteAddr())
		log.Printf("Broadcast content: '%s'", broadcast.Content)
	}

	// time.Sleep(time.Second * 2)
}

func sendUserMessage(message common.Broadcast) {
	for _, registeredUser := range UserList {
		if registeredUser.Connection != nil {
			encoder := gob.NewEncoder(registeredUser.Connection)
			err := encoder.Encode(message)
			if err != nil {
				log.Printf("Error while sending the message to the user '%s'\n", registeredUser.Username)
				continue
			}
			log.Printf("Broadcasted to user '%s'\n", registeredUser.Username)
			// registeredUser.Connection.Write([]byte(fmt.Sprintf("[%d]<%s> %s", message.Date, []byte(message.Sender), []byte(message.Content))))
		}
	}
}

func initialSetup(user *User) {
	reader := bufio.NewReader(user.Connection)
	for {
		log.Printf("ZONE 1")
		sendBroadcast(*user, common.Broadcast{
			Sender:    "__SERVER__",
			Content:   "Choose an username [4-15 chars]: ",
			Date:      time.Now().Unix(),
			Type:      common.MESSAGE,
			Printable: true,
			Code:      common.C_OK,
		})

		log.Printf("ZONE 2")
		desiredUsername, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error on reading username: '%s'\n", err.Error())
			user.Connection.Close()
			return
		}

		log.Printf("ZONE 3")
		desiredUsername = strings.TrimSpace(desiredUsername)

		if !(len(desiredUsername) >= 4 && len(desiredUsername) <= 15) {
			sendBroadcast(*user, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   "Incorrect. Username must be 4 or longer and 15 or less characters length.",
				Date:      time.Now().Unix(),
				Type:      common.ERROR,
				Printable: true,
				Code:      common.C_OK,
			})
			continue
		}

		user.Username = desiredUsername
		log.Printf("ZONE 4")
		break
	}
	log.Printf("ZONE 5")
	for {
		log.Printf("ZONE 6")
		sendBroadcast(*user, common.Broadcast{
			Sender:    "__SERVER__",
			Content:   "Choose a password [>5 chars]: ",
			Date:      time.Now().Unix(),
			Type:      common.MESSAGE,
			Printable: true,
			Code:      common.C_OK,
		})
		log.Printf("ZONE 7")

		desiredPassword, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error on reading username: '%s'\n", err.Error())
			user.Connection.Close()
			return
		}

		log.Printf("ZONE 8")
		desiredPassword = strings.TrimSpace(desiredPassword)

		if len(desiredPassword) < 5 {
			sendBroadcast(*user, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   "Incorrect. Password must be 5 characters length or greater.",
				Date:      time.Now().Unix(),
				Type:      common.MESSAGE,
				Printable: true,
				Code:      common.C_OK,
			})
			continue
		}

		user.Password = desiredPassword
		break
	}

	return
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	connUser := User{Username: "____UNDEFINED____", Password: "", Connection: conn, Encoder: gob.NewEncoder(conn)}

	sendBroadcast(connUser, common.Broadcast{
		Sender:    "__SERVER__",
		Content:   "🎉 Connected to the IRC server",
		Date:      time.Now().Unix(),
		Type:      common.MESSAGE,
		Printable: true,
		Code:      common.C_OK,
	})

	reader := bufio.NewReader(conn)
	for {

		initialSetup(&connUser)

		UserList = append(UserList, &connUser)

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

				broadcastMsg := common.Broadcast{
					Sender:    connUser.Username,
					Content:   msgContent,
					Date:      time.Now().Unix(),
					Type:      common.MESSAGE,
					Printable: true,
					Code:      common.C_OK,
				}

				go sendUserMessage(broadcastMsg)

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
