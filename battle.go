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

type Update struct {
	Player1Life	int `json:"p1life"`
	Player1Stam	float32 `json:"p1stam"`
	Player1State	string `json:"p1state"`
	Player1StateDur int `json:"p1stateDur"`
	Player1BlockDur int `json:"p1blockDur"`
	Player2Life	int `json:"p2life"`
	Player2Stam	float32 `json:"p2stam"`
	Player2State	string `json:"p2state"`
	Player2StateDur int `json:"p2stateDur"`
	Player2BlockDur int `json:"p2blockDur"`
}

func battle(player1socket, player2socket *websocket.Conn) {
	player1 := Player{Socket: player1socket, Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0}
	player2 := Player{Socket: player2socket, Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0}
	player1.Socket.WriteJSON(Message{Username:"",Content:"",Command:"START GAME"})
	player2.Socket.WriteJSON(Message{Username:"",Content:"",Command:"START GAME"})

}
