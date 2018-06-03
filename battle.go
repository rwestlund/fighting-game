package main

import (
	"github.com/gorilla/websocket"
	//	"log"
	//	"net/http"
)

type Player struct {
	Socket        *websocket.Conn
	Life          int
	Stamina       float32
	State         string
	StateDuration int
	BlockDuration int
}

func battle(player1socket, player2socket *websocket.Conn) {
	player1 := Player{Socket: player1socket, Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0}
	player2 := Player{Socket: player2socket, Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0}
	// These lines are just here to use the player structs so the code will compile.
	player1.Life = 100
	player2.Life = 100

}
