package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Message struct {
	Username string	`json:"username"`
	Content  string `json:"message"`
}

type Player struct {
	Socket *websocket.Conn
	Ready  bool
}

var players = make(map[Player]bool)
var chatchan = make(chan Message)
var upgrader = websocket.Upgrader{}

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
		for player := range players {
			err := player.Socket.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				player.Socket.Close()
				delete(players, player)
			}
		}
	}
}

func matchmaker() {
	for {
		readyPlayers := make([]*Player, 0)
		for player, _ := range(players) {
			if player.Ready==true {readyPlayers = append(readyPlayers, &player)}
		}
		log.Println(readyPlayers)
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
        // Upgrade initial GET request to a websocket
        socket, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
        	log.Fatal(err)
        }
        defer socket.Close()
        newPlayer:=Player{socket,false}
        players[newPlayer] = true
        for {
                var msg Message
		// Read the next message from chat
                err := socket.ReadJSON(&msg)
		log.Println("message:",msg)
		if msg.Content=="READY" {
			if !newPlayer.Ready {
				newPlayer.Ready=true
			} else {
				newPlayer.Ready=false
			}
			continue
		}
                if err != nil {
                        log.Printf("error: %v", err)
                        delete(players, newPlayer)
                        break
                }
                chatchan <- msg
        }
}