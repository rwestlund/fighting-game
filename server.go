package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type Message struct {
	Username string `json:"username"`
	Content  string `json:"message"`
	Command  string `json:"command"`
}

type User struct {
	Ready bool
}

var users = make(map[*websocket.Conn]*User)
var chatchan = make(chan Message)
var upgrader = websocket.Upgrader{}
var mutex = sync.Mutex{}

func main() {
	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnection)
	port := ":8000"
	log.Println("http server starting on port", port)
	go globalChat()
	go matchmaker()
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func globalChat() {
	for {
		msg := <-chatchan
		mutex.Lock()
		for socket, _ := range users {
			err := socket.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				socket.Close()
				delete(users, socket)
			}
		}
		mutex.Unlock()

	}
}

func matchmaker() {
	for {
		time.Sleep(1 * time.Second)
		readyUsers := make([]*websocket.Conn, 0)
		mutex.Lock()
		for socket, user := range users {
			if user.Ready == true {
				readyUsers = append(readyUsers, socket)
			}
		}
		if len(readyUsers) >= 2 {
			users[readyUsers[0]].Ready = false
			users[readyUsers[1]].Ready = false
			go battle(readyUsers[0], readyUsers[1])
		}
		mutex.Unlock()
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()
	mutex.Lock()
	newUser := User{false}
	users[socket] = &newUser
	mutex.Unlock()
	for {
		var msg Message
		// Read the next message from chat
		err := socket.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			mutex.Lock()
			delete(users, socket)
			mutex.Unlock()
			break
		}
		if msg.Command == "READY" {
			newUser.Ready = true
			continue
		}
		if msg.Command == "UNREADY" {
			newUser.Ready = false
			continue
		}
		if msg.Command == "START GAME" {
			continue
		}
		chatchan <- msg
	}
}
