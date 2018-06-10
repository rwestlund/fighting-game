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
	Ready  bool
	InGame bool
	Mutex  sync.Mutex
}

var users = make(map[*websocket.Conn]*User)
var chatchan = make(chan Message)
var upgrader = websocket.Upgrader{}
var global_mutex = sync.Mutex{}

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
		global_mutex.Lock()
		for socket, _ := range users {
			err := socket.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				socket.Close()
				delete(users, socket)
			}
		}
		global_mutex.Unlock()

	}
}

func matchmaker() {
	for {
		time.Sleep(1 * time.Second)
		readyUsers := make([]*websocket.Conn, 0)
		global_mutex.Lock()
		for socket, user := range users {
			user.Mutex.Lock()
			if user.Ready == true {
				readyUsers = append(readyUsers, socket)
			}
			user.Mutex.Unlock()
		}
		if len(readyUsers) >= 2 {
			users[readyUsers[0]].Mutex.Lock()
			users[readyUsers[0]].Ready = false
			users[readyUsers[0]].InGame = true
			users[readyUsers[0]].Mutex.Unlock()
			users[readyUsers[1]].Mutex.Lock()
			users[readyUsers[1]].Ready = false
			users[readyUsers[1]].InGame = true
			users[readyUsers[1]].Mutex.Unlock()
			go battle(readyUsers[0], readyUsers[1])
		}
		global_mutex.Unlock()
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()
	global_mutex.Lock()
	newUser := User{false, false, sync.Mutex{}}
	users[socket] = &newUser
	global_mutex.Unlock()
	var msg Message
	for {
		newUser.Mutex.Lock()
		for newUser.InGame {
			time.Sleep(1 * time.Second)
		}
		err := socket.ReadJSON(&msg)
		newUser.Mutex.Unlock()
		// Read the next message from chat
		log.Println("server.go trying to readJSON...")
		log.Println("server.go got message:", msg)
		if err != nil {
			// This continue statement is here because this error is caused by the leftover
			// messages that got stuck due to the mutex blocking getting read finally when
			// the battle starts. It's not a fatal error, so I didn't want it crashing the
			// program.
			if err.Error() == "invalid character 'N' looking for beginning of value" {
				continue
			}
			log.Printf("error: %v", err)
			global_mutex.Lock()
			delete(users, socket)
			global_mutex.Unlock()
			break
		}
		if msg.Command == "READY" {
			newUser.Mutex.Lock() // I don't know if these are necessary
			newUser.Ready = true
			newUser.Mutex.Unlock()
			continue
		}
		if msg.Command == "UNREADY" {
			newUser.Mutex.Lock() // I don't know if these are necessary
			newUser.Ready = false
			newUser.Mutex.Unlock()
			continue
		}
		if msg.Command == "START GAME" {
			continue
		}
		newUser.Mutex.Lock() // I don't know if these are necessary
		if newUser.InGame {
			continue
		}
		newUser.Mutex.Unlock()
		chatchan <- msg
	}
}
