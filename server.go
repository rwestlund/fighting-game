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
	Ready            bool
	InGame           bool
	BattleInputChan  chan Message
	BattleUpdateChan chan Update
	Mutex            sync.Mutex
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
	readyUsers := make([]*websocket.Conn, 0)
	global_mutex.Lock()
	for socket, user := range users {
		user.Mutex.Lock()
		if user.Ready == true {
			readyUsers = append(readyUsers, socket)
		}
		user.Mutex.Unlock()
	}
	log.Println("ready users:", len(readyUsers))
	if len(readyUsers) >= 2 {
		users[readyUsers[0]].Mutex.Lock()
		users[readyUsers[0]].Ready = false
		users[readyUsers[0]].InGame = true
		users[readyUsers[1]].Mutex.Lock()
		users[readyUsers[1]].Ready = false
		users[readyUsers[1]].InGame = true
		readyUsers[0].WriteJSON(Message{Username: "", Content: "", Command: "START GAME"})
		readyUsers[1].WriteJSON(Message{Username: "", Content: "", Command: "START GAME"})
		go battle(users[readyUsers[0]].BattleInputChan, users[readyUsers[1]].BattleInputChan, users[readyUsers[0]].BattleUpdateChan, users[readyUsers[1]].BattleUpdateChan)
		go forwardUpdates(readyUsers[0], users[readyUsers[0]].BattleUpdateChan)
		go forwardUpdates(readyUsers[1], users[readyUsers[1]].BattleUpdateChan)
		users[readyUsers[0]].Mutex.Unlock()
		users[readyUsers[1]].Mutex.Unlock()
	}
	global_mutex.Unlock()
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()
	global_mutex.Lock()
	newUser := User{false, false, make(chan Message), make(chan Update), sync.Mutex{}}
	users[socket] = &newUser
	global_mutex.Unlock()
	var msg Message
	for {
		// Read the next message from chat
		err := socket.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			global_mutex.Lock()
			delete(users, socket)
			global_mutex.Unlock()
			break
		}
		newUser.Mutex.Lock()
		if newUser.InGame {
			newUser.BattleInputChan <- msg
			newUser.Mutex.Unlock()
			continue
		}
		newUser.Mutex.Unlock()

		if msg.Command == "READY" {
			newUser.Mutex.Lock()
			newUser.Ready = true
			newUser.Mutex.Unlock()
			matchmaker()
			continue
		}
		if msg.Command == "UNREADY" {
			newUser.Mutex.Lock()
			newUser.Ready = false
			newUser.Mutex.Unlock()
			continue
		}
		chatchan <- msg
	}
}

func forwardUpdates(socket *websocket.Conn, channel chan Update) {
	for true {
		update := <-channel
		//		log.Println("forwardUpdates here, got an update:",update)
		socket.WriteJSON(update)
	}
}
