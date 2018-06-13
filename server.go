package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

// Messages are the JSON objects used for most communication between clients and the server. In the case of a chat message, Content will be used and Command will be left blank. In the context of a special message, Command will be used and Content will be left blank.
type Message struct {
	Username string `json:"username"`
	Content  string `json:"message"`
	Command  string `json:"command"`
}

// The two channels in this struct are for the player sending commands to the server and for the server sending gamestate updates to the player's computer.
type User struct {
	Ready            bool
	InGame           bool
	BattleInputChan  chan Message
	BattleUpdateChan chan Update
	Mutex            sync.Mutex
}

func main() {
	var users = make(map[*websocket.Conn]*User)
	var chatchan = make(chan Message)
	// The global mutex protects the users map from concurrency issues.
	var globalMutex = sync.Mutex{}

	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)
	// handleConnection actually returns an anonymous function that handles connections.
	http.HandleFunc("/ws", handleConnection(users, chatchan, globalMutex))
	port := ":8000"
	log.Println("http server starting on port", port)
	go globalChat(users, chatchan, globalMutex)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// This function listens for new chat messages and writes them out to each connected user.
func globalChat(users map[*websocket.Conn]*User, chatchan chan Message, globalMutex sync.Mutex) {
	for {
		msg := <-chatchan
		globalMutex.Lock()
		for socket := range users {
			err := socket.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				socket.Close()
				delete(users, socket)
			}
		}
		globalMutex.Unlock()

	}
}

// This function is called whenever a new player readies for battle. If at least two people are ready for battle, it matches two of them.
func matchmaker(users map[*websocket.Conn]*User, globalMutex sync.Mutex) {
	readyUsers := make([]*websocket.Conn, 0)
	globalMutex.Lock()
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
	globalMutex.Unlock()
}

// Each time a new user connects, a goroutine running the function that this one returns is created. It keeps track of the connection and sends chat data or game data back and forth.
func handleConnection(users map[*websocket.Conn]*User, chatchan chan Message, globalMutex sync.Mutex) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade initial GET request to a websocket
		var upgrader = websocket.Upgrader{}
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer socket.Close()
		globalMutex.Lock()
		newUser := User{false, false, make(chan Message), make(chan Update), sync.Mutex{}}
		users[socket] = &newUser
		globalMutex.Unlock()
		var msg Message
		for {
			// Read the next message from chat
			err := socket.ReadJSON(&msg)
			if err != nil {
				log.Printf("error: %v", err)
				globalMutex.Lock()
				delete(users, socket)
				globalMutex.Unlock()
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
				matchmaker(users, globalMutex)
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
}

// This goroutine listens for gamestate updates from battle.go and forwards them to the player.
func forwardUpdates(socket *websocket.Conn, channel chan Update) {
	for true {
		update := <-channel
		socket.WriteJSON(update)
	}
}
