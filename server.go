package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type Message struct {
	Username string `json:"username"`
	Content  string `json:"message"`
	Command  string `json:"command"`
}

type User struct {
	Socket *websocket.Conn
	Ready  bool
}



var users = make(map[*User]bool)
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
		for user, _ := range users {
			err := user.Socket.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				user.Socket.Close()
				delete(users, user)
			}
		}
		mutex.Unlock()

	}
}

func matchmaker() {
	for {
		readyUsers := make([]*User, 0)
		mutex.Lock()
		for user, _ := range users {
			if user.Ready == true {
				readyUsers = append(readyUsers, user)
			}
		}
		//		log.Println(readyUsers)
		if len(readyUsers) >= 2 {
			readyUsers[0].Ready = false
			readyUsers[1].Ready = false
			go battle(readyUsers[0].Socket, readyUsers[1].Socket)
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
	newUser := User{socket, false}
	users[&newUser] = true
	mutex.Unlock()
	for {
		var msg Message
		// Read the next message from chat
		err := socket.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			mutex.Lock()
			delete(users, &newUser)
			mutex.Unlock()
			break
		}
		log.Println("message:", msg)
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
