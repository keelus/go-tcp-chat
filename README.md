<h1 align="center">go-tcp-chat</h1>

<p align="center">
  <a href="./LICENSE.md"><img src="https://img.shields.io/badge/⚖️ license-MIT-blue" alt="MIT License"></a>
  <img src="https://img.shields.io/github/stars/keelus/go-tcp-chat?color=red&logo=github" alt="stars">
</p>

## ℹ️ Description
A TCP Chat implementation in Golang. Chat from multiple clients in the same (local/remote) server!
> In development

## 👋 Commands
- `/login <username> <password>` - Use it to log in in your account.
- `/register <username> <password>` - Use it to register a new account.
- `/msg <some text>` - Send something to the others!
- `/quit` - Log out from the chat.
- `/help` - Show available commands   

## ▶️ Run it
First, run the server:
```bash
go run ./server -ip=X -port=Y # Optionally, you can specify the IP and port [default: 127.0.0.1:6969]
```
Then, while the server is running, run as much clients as you want:
```bash
go run ./client -ip=X -port=Y # Optionally, you can specify the IP and port [default: 127.0.0.1:6969]
```

## ⚖️ License
This project is open source under the terms of the [MIT License](./LICENSE)

<br />
Made by <a href="https://github.com/keelus">keelus</a> ✌️
