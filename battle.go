package main

import (
	"log"
	"time"
)

// These two channels are the same as the two from the corresponding User struct in server.go.
// The Command field is the current input from the player (kept up to date by the concurrently running input func for the player).
// The state field keeps track of what the player is doing. It has values like "standing", "blocking", "light attack", etc.
// The StateDuration field shows how much longer the player will remain in their current state.
// The BlockDuration field tells how long has the player has been blocking for. This is used to tell whether the player blocked reactively to counter an attack as opposed to just holding down block before it started.
// The Finished field shows what state the player just exited. It's used to know when an attack is supposed to land.
type Player struct {
	InputChan     chan Message
	UpdateChan    chan Update
	Command       string
	Life          int
	Stamina       float32
	State         string
	StateDuration int
	BlockDuration int
	Finished      string
}

// This struct is passed instead of Player to the client in Updates so that unneeded fields like the channels aren't passed.
type PlayerStatus struct {
	Life          int     `json:"life"`
	Stamina       float32 `json:"stamina"`
	State         string  `json:"state"`
	StateDuration int     `json:"stateDur"`
}

func (p *Player) Status() PlayerStatus {
	return PlayerStatus{Life: p.Life, Stamina: p.Stamina, State: p.State, StateDuration: p.StateDuration}

}

// This is called every mainloop cycle, and does two things: regenerate stamina, and make progress toward exiting the current state.
func (p *Player) PassTime(amount int) {
	p.Stamina += 0.1
	if p.Stamina > 100 {
		p.Stamina = 100
	}
	p.StateDuration -= amount
	if p.StateDuration <= 0 {
		p.StateDuration = 0
		p.Finished = p.State
		p.State = "standing"
	}
}

func (p *Player) SetState(state string, duration int) {
	p.State = state
	p.StateDuration = duration
}

// One of these is sent back to each player every mainloop cycle. Note that the players don't know which player they are internally - it doesn't matter.
type Update struct {
	Self  PlayerStatus `json:"self"`
	Enemy PlayerStatus `json:"enemy"`
}

func battle(player1inputChan, player2inputChan chan Message, player1updateChan, player2updateChan chan Update) {
	log.Println("in battle")
	// constants
	const LIGHT_ATTACK_DMG int = 3
	const LIGHT_ATTACK_COST float32 = 5.0
	const LIGHT_ATTACK_BLK_COST float32 = 5.0
	const LIGHT_ATTACK_SPD int = 50
	players := []*Player{&Player{InputChan: player1inputChan, UpdateChan: player1updateChan, Command: "NONE", Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0, Finished: ""}, &Player{InputChan: player2inputChan, UpdateChan: player2updateChan, Command: "NONE", Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0, Finished: ""}}
	go input(players[0])
	go input(players[1])

	for players[0].Life > 0 && players[1].Life > 0 {
		log.Println("mainloop")
		time.Sleep(100 * time.Millisecond)
		players[0].UpdateChan <- Update{Self: players[0].Status(), Enemy: players[1].Status()}
		players[1].UpdateChan <- Update{Self: players[1].Status(), Enemy: players[0].Status()}
		// Do the backend stuff.
		for _, player := range players {
			player.PassTime(1)
			// Get input from the players.
			switch player.Command {
			case "LIGHT":
				player.Command = "NONE"
				if player.Stamina >= LIGHT_ATTACK_COST {
					player.SetState("light attack", LIGHT_ATTACK_SPD)
					player.Stamina -= LIGHT_ATTACK_COST
				}
			}
		}
	}

}

// This function listens continuously for an input from the player and passes it through to the player's Command field, where the mainloop can see it.
func input(player *Player) {
	for true {
		player.Command = (<-player.InputChan).Content
	}
}
