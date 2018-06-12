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
	OwnLife       int     `json:"ownLife"`
	OwnStam       float32 `json:"ownStam"`
	OwnState      string  `json:"ownState"`
	OwnStateDur   int     `json:"ownStateDur"`
	OwnBlockDur   int     `json:"ownBlockDur"`
	EnemyLife     int     `json:"enemyLife"`
	EnemyStam     float32 `json:"enemyStam"`
	EnemyState    string  `json:"enemyState"`
	EnemyStateDur int     `json:"enemyStateDur"`
	EnemyBlockDur int     `json:"enemyBlockDur"`
}

func battle(player1inputChan, player2inputChan chan Message, player1updateChan, player2updateChan chan Update) {
	log.Println("in battle")
	//	global_mutex.Lock()
	//	defer global_mutex.Unlock()
	// constants
	const LIGHT_ATTACK_DMG int = 3
	const LIGHT_ATTACK_COST float32 = 5.0
	const LIGHT_ATTACK_BLK_COST float32 = 5.0
	const LIGHT_ATTACK_SPD int = 50
	player1 := Player{InputChan: player1inputChan, UpdateChan: player1updateChan, Command: "NONE", Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0, Finished: ""}
	player2 := Player{InputChan: player2inputChan, UpdateChan: player2updateChan, Command: "NONE", Life: 100, Stamina: 100, State: "standing", StateDuration: 0, BlockDuration: 0, Finished: ""}
	players := []*Player{&player1, &player2}
	go input(&player1)
	go input(&player2)

	for player1.Life > 0 && player2.Life > 0 {
		time.Sleep(1000 * time.Millisecond)
		player1.UpdateChan <- Update{OwnLife: player1.Life, OwnStam: player1.Stamina, OwnState: player1.State, OwnStateDur: player1.StateDuration, OwnBlockDur: player1.BlockDuration, EnemyLife: player2.Life, EnemyStam: player2.Stamina, EnemyState: player2.State, EnemyStateDur: player2.StateDuration, EnemyBlockDur: player2.BlockDuration}
		player2.UpdateChan <- Update{OwnLife: player2.Life, OwnStam: player2.Stamina, OwnState: player2.State, OwnStateDur: player2.StateDuration, OwnBlockDur: player2.BlockDuration, EnemyLife: player1.Life, EnemyStam: player1.Stamina, EnemyState: player1.State, EnemyStateDur: player1.StateDuration, EnemyBlockDur: player1.BlockDuration}
		// Do the backend stuff.
		for _, player := range players {
			player.PassTime(1)
		}
		// Get input from the players.
		for _, player := range players {
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
		command := <-player.InputChan
		log.Println("input() printing player command:", command)
		player.Command = command.Command
	}
}
