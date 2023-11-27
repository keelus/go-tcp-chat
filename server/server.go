package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"go-tcp-chat/common"
	"log"
	"net"
	"strings"
	"time"
)

const SERVER_VERSION = common.TCP_CHAT_VERSION

var UserList []*User = make([]*User, 0)

func sendBroadcast(user *User, broadcast common.Broadcast) bool {
	broadcast.SentFrom = common.L_SERVER
	broadcast.Date = time.Now()

	err := user.Encoder.Encode(broadcast)
	if err != nil {
		if netErr, ok := err.(*net.OpError); ok && !netErr.Temporary() { // Client's connection was closed but status was not updated (client exited without /quit)
			user.Connection = nil
			user.Encoder = nil
			log.Printf("[INFO] The %s' old connection was terminated\n", user.Username)

			if user.Logged {
				sendGlobalBroadcast(common.Broadcast{
					Sender:    "__SERVER__",
					Content:   fmt.Sprintf("%s's connection timed out.", user.Username),
					Type:      common.ACTIVITY,
					Printable: true,
					Code:      common.C_OK,
				})
			}

			return false
		}

		log.Printf("[ERROR] Error sending a broadcast\n\tAddress: %s\n\tReason: %s\n", user.Connection.RemoteAddr(), err.Error())
		return false
	}

	return true
}

func sendGlobalBroadcast(message common.Broadcast) {
	for _, registeredUser := range UserList {
		if registeredUser.Connection != nil {
			sendBroadcast(registeredUser, message)
		}
	}
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
		Content:   fmt.Sprintf("Connection stablished. Server running on version %s", SERVER_VERSION),
		Type:      common.TEXT,
		Printable: true,
		Code:      common.C_OK,
	})

	sendBroadcast(&connUser, common.Broadcast{
		Sender:    "__SERVER__",
		Content:   SERVER_VERSION,
		Type:      common.VERSION,
		Printable: false,
		Code:      common.C_OK,
	})

	// Content:   fmt.Sprintf("You need to use the same version as the server. \n   Server version:%s\n   Your (client) version: %s", SERVER_VERSION, client_version),

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
			log.Printf("[ERROR] Error reading message: '%s'\n", err.Error())
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
				Content:   fmt.Sprintf("Logged as %s. Welcome!", username),
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})
			sendGlobalBroadcast(common.Broadcast{
				Sender:    "__SERVER__",
				Content:   fmt.Sprintf("%s has joined the chat.", connUser.Username),
				Type:      common.ACTIVITY,
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
				Content:   fmt.Sprintf("Registered and logged as %s. Welcome!", username),
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})

			sendGlobalBroadcast(common.Broadcast{
				Sender:    "__SERVER__",
				Content:   fmt.Sprintf("%s has joined the chat.", connUser.Username),
				Type:      common.ACTIVITY,
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

			log.Printf("[INFO] %s broadcasted.\n", connUser.Username)

			msgContent := strings.Replace(msg, "/msg", "", 1)
			msgContent = fmt.Sprintf("%s\n", msgContent)

			broadcastMsg := common.Broadcast{
				Sender:    connUser.Username,
				Content:   msgContent,
				Type:      common.MESSAGE,
				Printable: true,
				Code:      common.C_OK,
			}

			go sendGlobalBroadcast(broadcastMsg)

			break
		case "/quit":
			sendBroadcast(&connUser, common.Broadcast{
				Sender:    "__SERVER__",
				Content:   "Goodbye!",
				Type:      common.TEXT,
				Printable: true,
				Code:      common.C_OK,
			})

			if connUser.Logged {
				log.Printf("[INFO] User %s[%s] disconnected.\n", connUser.Username, connUser.Connection.RemoteAddr())
				connUser.Connection = nil
				connUser.Encoder = nil

				sendGlobalBroadcast(common.Broadcast{
					Sender:    "__SERVER__",
					Content:   fmt.Sprintf("%s has left the chat.", connUser.Username),
					Type:      common.ACTIVITY,
					Printable: true,
					Code:      common.C_OK,
				})
			} else {
				log.Printf("[INFO] A non logged user[%s] disconnected.\n", connUser.Connection.RemoteAddr())
			}

			return
		case "/help":
			sendBroadcast(&connUser, common.Broadcast{
				Sender: "__SERVER__",
				// Content:   fmt.Sprintf("Available commands:\n\t/login <username> <password>\n\t/register <username> <password>\n\t/msg <message content>\n\t/quit\n\t/help"),
				Content:   "Available commands:\n   /login <username> <password> - Use it to log in in your account.\n   /register <username> <password> - Use it to register a new account.\n   /msg <some text> - Send something to the others!\n   /quit - Log out from the chat.\n   /help - Show available commands",
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
	argIp := flag.String("ip", "127.0.0.1", "The local IP")
	argPort := flag.String("port", "6969", "The port")
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", *argIp, *argPort))
	if err != nil {
		log.Fatalf("[ERROR] An error happened while listening to the port %s. Reason:'%s'\n", *argPort, err.Error())
	}
	defer listener.Close()

	log.Printf("[INFO] Listening TCP in %s:%s...\n", *argIp, *argPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[ERROR] There was an error handling the connection from %s.\n", conn.RemoteAddr())
		} else {
			log.Printf("[INFO] Client [%s] connected.", conn.RemoteAddr())
			go handleConnection(conn)
		}
	}
}
