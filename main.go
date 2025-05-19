package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

const (
	HOST = "0.0.0.0"
	PORT = "1234"
)

var COLORS = map[string]string{
	"red":     "\033[91m",
	"green":   "\033[92m",
	"yellow":  "\033[93m",
	"blue":    "\033[94m",
	"magenta": "\033[95m",
	"cyan":    "\033[96m",
}
const RESET = "\033[0m"

type Client struct {
	conn     net.Conn
	username string
	color    string
}

var clients []Client
var mutex = &sync.Mutex{}

func broadcast(sender *Client, message string) {
	mutex.Lock()
	defer mutex.Unlock()
	for _, c := range clients {
		if c.conn != sender.conn {
			c.conn.Write([]byte("\r\033[K" + message + "\rYou > "))
		}
	}
}

func listOnlineUsers(client *Client) string {
	mutex.Lock()
	defer mutex.Unlock()
	names := []string{}
	for _, c := range clients {
		var isMe string = ""
		if c.conn == client.conn {
			isMe = " (You)"
		}
		names = append(names, fmt.Sprintf("%s%s%s%s", c.color, c.username, RESET, isMe))
	}
	return strings.Join(names, ", ")
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	conn.Write([]byte("Welcome to TCP Chat!\nLogin as [1] Guest or [2] Username? (1/2): "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var username string
	if choice == "2" {
		conn.Write([]byte("Enter your username: "))
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	} else {
		username = fmt.Sprintf("Guest%d", conn.RemoteAddr().(*net.TCPAddr).Port)
	}

	// Color selection
	conn.Write([]byte("\nAvailable colors: " + strings.Join(getColorNames(), ", ") + "\n"))
	var color string
	for {
		conn.Write([]byte("Choose your name color: "))
		color, _ = reader.ReadString('\n')
		color = strings.ToLower(strings.TrimSpace(color))
		if _, ok := COLORS[color]; ok {
			break
		}
		conn.Write([]byte("Invalid color. Try again.\n"))
	}

	client := Client{
		conn:     conn,
		username: username,
		color:    COLORS[color],
	}

	mutex.Lock()
	clients = append(clients, client)
	mutex.Unlock()

	broadcast(&client, fmt.Sprintf("%s joined the chat.\n", username))
	conn.Write([]byte(fmt.Sprintf("\nType /quit to leave.\n%sYou > %s", client.color, RESET)))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

		switch {
		case line == "/quit":
			return

		case line == "/whoami":
			conn.Write([]byte(fmt.Sprintf("\r\033[KYou are: %s%s%s\nYou > ", client.color, client.username, RESET)))

		case line == "/online" || line == "/all":
			online := listOnlineUsers(&client)
			conn.Write([]byte("\r\033[KOnline: " + online + "\nYou > "))

		case strings.HasPrefix(line, "/"):
			conn.Write([]byte("\r\033[KUnknown command.\nYou > "))

		default:
			message := fmt.Sprintf("%s%s%s > %s\n", client.color, client.username, RESET, line)
			broadcast(&client, message)
			conn.Write([]byte(fmt.Sprintf("\r%sYou > %s", client.color, RESET)))
		}
	}

	// Cleanup
	mutex.Lock()
	for i, c := range clients {
		if c.conn == conn {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	mutex.Unlock()
	broadcast(&client, fmt.Sprintf("%s left the chat.\n", username))
}

func getColorNames() []string {
	names := make([]string, 0, len(COLORS))
	for k := range COLORS {
		names = append(names, k)
	}
	return names
}

func main() {
	ln, err := net.Listen("tcp", HOST+":"+PORT)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	fmt.Printf("[+] Telnet Chat Server running on %s:%s\n", HOST, PORT)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err)
			continue
		}
		go handleClient(conn)
	}
}
