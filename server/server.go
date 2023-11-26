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

func sendBroadcast(user *User, broadcast common.Broadcast) bool {
	broadcast.SentFrom = common.L_SERVER
	broadcast.Date = time.Now().Unix()

	err := user.Encoder.Encode(broadcast)
	if err != nil {
		if netErr, ok := err.(*net.OpError); ok && !netErr.Temporary() { // Client's connection was closed but status was not updated (client exited without /quit)
			user.Connection = nil
			user.Encoder = nil
			log.Printf("%s' old connection was terminated\n", user.Username)
			return false
		}

		log.Printf("Error sending a broadcast\n\tAddress: %s\n\tReason: %s\n", user.Connection.RemoteAddr(), err.Error())
		return false
	}

	return true
}

func sendUserMessage(message common.Broadcast) {
	for _, registeredUser := range UserList {
		if registeredUser.Connection != nil {
			sendBroadcast(registeredUser, message)
		}
	}
}

func initialSetup(user *User) error {
	reader := bufio.NewReader(user.Connection)
	for {
		sendBroadcast(user, common.Broadcast{
			Sender:    "__SERVER__",
			Content:   "Choose an username [4-15 chars]: ",
			Type:      common.TEXT,
			Printable: true,
			Code:      common.C_OK,
		})

		desiredUsername, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error on reading username: '%s'\n", err.Error())
			user.Connection.Close()
			return err
		}

		desiredUsername = strings.TrimSpace(desiredUsername)

		if !(len(desiredUsername) >= 4 && len(desiredUsername) <= 15) {
			sendBroadcast(user, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   "Incorrect. Username must be 4 or longer and 15 or less characters length.",
				Type:      common.ERROR,
				Printable: true,
				Code:      common.C_OK,
			})
			continue
		}

		user.Username = desiredUsername
		break
	}

	for {
		sendBroadcast(user, common.Broadcast{
			Sender:    "__SERVER__",
			Content:   "Choose a password [>5 chars]: ",
			Type:      common.TEXT,
			Printable: true,
			Code:      common.C_OK,
		})

		desiredPassword, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error on reading username: '%s'\n", err.Error())
			user.Connection.Close()
			return err
		}

		desiredPassword = strings.TrimSpace(desiredPassword)

		if len(desiredPassword) < 5 {
			sendBroadcast(user, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   "Incorrect. Password must be 5 characters length or greater.",
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})
			continue
		}

		user.Password = desiredPassword
		break
	}

	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	connUser := User{
		Username:   "____UNDEFINED____",
		Password:   "",
		Connection: conn,
		Encoder:    gob.NewEncoder(conn), // Create a new encoder with the new connection
		Logged:     false,
	}

	sendBroadcast(&connUser, common.Broadcast{
		Sender:    "__SERVER__",
		Content:   "🎉 Connected to the IRC server",
		Type:      common.TEXT,
		Printable: true,
		Code:      common.C_OK,
	})

	sendBroadcast(&connUser, common.Broadcast{
		Sender:    "__SERVER__",
		Content:   "Login with your account or create a new one. Type /help to see available commands.",
		Type:      common.TEXT,
		Printable: true,
		Code:      common.C_OK,
	})

	reader := bufio.NewReader(conn)
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
		case "/login": // /login <username> <password>
			if len(msgParts) != 3 {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "Invalid usage. Command usage: /login <username> <password>",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}
			if connUser.Logged {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "You are already logged in.",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}

			username := msgParts[1]
			password := msgParts[2]

			found := -1
			for i, user := range UserList {
				if user.Username == username && user.Password == password {
					found = i
				}
			}

			if found == -1 {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "Entered credentials are incorrect.",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}

			if UserList[found].Connection != nil {
				ok := sendBroadcast(UserList[found], common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "You logged in from another terminal. Closing this session.",
					Type:      common.TEXT,
					Printable: true,
					Code:      common.C_OK,
				})
				if ok {
					UserList[found].Connection.Close()
				}
			}

			connUser.Username = username
			connUser.Password = password
			connUser.Logged = true
			UserList[found] = &connUser

			sendBroadcast(&connUser, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   fmt.Sprintf("Logged as <%s>. Welcome!", username),
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})
			break
		case "/register": // /register <username> <password>
			if len(msgParts) != 3 {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "Invalid usage. Command usage: /register <username> <password>",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}
			if connUser.Logged {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "You are already logged in.",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}

			username := msgParts[1]
			password := msgParts[2]

			found := false
			for _, user := range UserList {
				if user.Username == username {
					found = true
				}
			}

			if found {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "That username is already in use.",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}

			connUser.Username = username
			connUser.Password = password
			connUser.Logged = true
			UserList = append(UserList, &connUser)

			sendBroadcast(&connUser, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   fmt.Sprintf("Registered and logged as <%s>. Welcome!", username),
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})
			break
		case "/msg":
			if !connUser.Logged {
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "You are not logged in. You can do so with /login. Type /help to show all commands.",
					Type:      common.ERROR,
					Printable: true,
					Code:      common.C_ERROR,
				})
				continue
			}

			log.Printf("%s sent a message\n", connUser.Username)

			msgContent := strings.Replace(msg, "/msg", "", 1)
			msgContent = fmt.Sprintf("%s\n", msgContent)

			broadcastMsg := common.Broadcast{
				Sender:    connUser.Username,
				Content:   msgContent,
				Type:      common.MESSAGE,
				Printable: true,
				Code:      common.C_OK,
			}

			go sendUserMessage(broadcastMsg)

			break
		case "/quit":
			if connUser.Logged {
				log.Printf("%s disconnected\n", connUser.Username)
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   fmt.Sprintf("Goodbye %s!", connUser.Username),
					Type:      common.TEXT,
					Printable: true,
					Code:      common.C_OK,
				})

				for _, user := range UserList {
					if user.Username == connUser.Username {
						user.Connection = nil
						user.Encoder = nil
					}
				}
			} else {
				log.Printf("%s disconnected\n", connUser.Connection.RemoteAddr())
				sendBroadcast(&connUser, common.Broadcast{
					Sender:    "__SERVER__",
					Content:   "Goodbye!",
					Type:      common.TEXT,
					Printable: true,
					Code:      common.C_OK,
				})
			}

			return
		case "/help":
			sendBroadcast(&connUser, common.Broadcast{
				Sender: "__SERVER__",
				// Content:   fmt.Sprintf("Available commands:\n\t/login <username> <password>\n\t/register <username> <password>\n\t/msg <message content>\n\t/quit\n\t/help"),
				Content:   "Available commands: /login, /register, /msg, /quit, /help",
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})
		default:
			sendBroadcast(&connUser, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   fmt.Sprintf("Unknown command '%s'. Type /help to view all commands.", msgParts[0]),
				Type:      common.ERROR,
				Printable: true,
				Code:      common.C_ERROR,
			})
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
