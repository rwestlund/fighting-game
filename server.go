package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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

var users = make(map[*websocket.Conn]*User)
var chatchan = make(chan Message)
var upgrader = websocket.Upgrader{}

// The global mutex protects the users map from concurrency issues.
var globalMutex = sync.Mutex{}

func main() {
	// When new clients arrive, their IO channels will be sent through here.
	var newClients = make(chan ConnInfo)

	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)
	http.Handle("/ws", handleConnection(newClients))
	port := ":8000"
	log.Println("http server starting on port", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// This function is called whenever a new player readies for battle. If at least two people are ready for battle, it matches two of them.
func matchmaker() {
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

// ConnInfo models the communication channel between a user's client and the
// server.
type ConnInfo struct {
	Inbound  chan Message
	Outbound chan Message
}

// MessageInfo wraps a Message with a reference to the User that sent it.
type MessageInfo struct {
	Message Message
	User    *User
}

// dispatcher takes a channel to receive new clients on and coordinates
// high-level message passing. It alone has the list of all connected clients,
// so no mutex is needed. Because it only takes in ConnInfos, it doesn't care
// how the clients are connected.
func dispatcher(newClients <-chan ConnInfo) {
	// The list of clients never leaves this scope.
	var clients = make(map[*ConnInfo]User)
	// All incoming messages will be merged into this channel.
	var messages = make(chan MessageInfo)
	// This is used for clients that disconnect, so they can be removed.
	var leaving = make(chan *ConnInfo)
	for {
		select {
		// When a new connection is established.
		case newConn := <-newClients:
			// Add them to the list.
			var user User
			clients[&newConn] = user

			// Merge their Messages ino the single messages channel.
			go func(sink chan<- MessageInfo, conn *ConnInfo, user *User,
				leaving chan<- *ConnInfo) {
				for m := range conn.Inbound {
					// Associate the Message with the User so we can tell who
					// sent it later.
					sink <- MessageInfo{Message: m, User: user}
				}
				// Let dispatch know that they're gone before we exit.
				leaving <- conn
			}(messages, &newConn, &user, leaving)

		// Delete clients when they disconnect.
		case oldConn := <-leaving:
			delete(clients, oldConn)

		// When a Message is received from anyone.
		case msg := <-messages:
			// If they're in a game, forward all messages there.
			if msg.User.InGame {
				msg.User.BattleInputChan <- msg.Message

				// Handle lobby command messages.
			} else if msg.Message.Command != "" {
				if msg.Message.Command == "READY" {
					msg.User.Ready = true
					// Try to start a match.
					matchmaker()
				} else if msg.Message.Command == "UNREADY" {
					msg.User.Ready = false
				}

				// Handle lobby chat messages.
			} else {
				for conn := range clients {
					conn.Outbound <- msg.Message
				}
			}
		}
	}
}

// Each time a new user connects, a goroutine running the returned function is created. It keeps track of the connection and sends chat data or game data back and forth.
func handleConnection(newClients chan<- ConnInfo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upgrade initial GET request to a websocket
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
		}
		defer socket.Close()
		// Send the connection info.
		var conn = ConnInfo{
			Inbound:  make(chan Message),
			Outbound: make(chan Message),
		}
		// This will let the consumer know that it's no longer active.
		defer close(conn.Inbound)
		defer close(conn.Outbound)
		// Signal that a new client has arrived.
		newClients <- conn

		// Connect the outbound channel to the websocket.
		go func() {
			for msg := range conn.Outbound {
				var err = socket.WriteJSON(msg)
				log.Println(err)
				//TODO remove them or just drop the message?
			}
		}()

		// Connect the websocket to the inbound channel.
		var msg Message
		for {
			// Read the next message from chat
			err := socket.ReadJSON(&msg)
			if err != nil {
				log.Printf("error: %v", err)
				return
			}
			conn.Inbound <- msg
		}
	})
}

// This goroutine listens for gamestate updates from battle.go and forwards them to the player.
func forwardUpdates(socket *websocket.Conn, channel chan Update) {
	for true {
		update := <-channel
		socket.WriteJSON(update)
	}
}
