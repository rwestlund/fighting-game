package main

import (
	"log"
	"time"
)

// These two channels are the same as the two from the corresponding User struct in server.go.
// The Command field is the current input from the player (kept up to date by the concurrently running input func for the player).
// The state field keeps track of what the player is doing. It has values like "standing", "blocking", "light attack", etc.
// The StateDuration field shows how much longer the player will remain in their current state.
// The Finished field shows what state the player just exited. It's used to know when an attack is supposed to land.
type Player struct {
	Name          string
	InputChan     chan Message
	UpdateChan    chan Update
	Command       string
	Life          int
	Stamina       float32
	State         string
	StateDuration int
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
//	p.Stamina += 0.1
	if p.Stamina > 100 {
		p.Stamina = 100
	}
	p.StateDuration -= amount
	if p.StateDuration <= 0 && !TERMINAL_STATES[p.State] {
		log.Println("state", p.State, "timed out")
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

// constants
const LIGHT_ATK_DMG int = 3
const LIGHT_ATK_SPD int = 50
const LIGHT_ATK_COST float32 = 10.0
const LIGHT_ATK_BLK_COST float32 = 8.0
const LIGHT_ATK_CNTR_SPD int = 30
const LIGHT_ATK_CNTR_DMG int = 3
const LIGHT_ATK_CNTR_SAVE_COST float32 = 10.0
const HEAVY_ATK_DMG int = 7
const HEAVY_ATK_SPD int = 80
const HEAVY_ATK_COST float32 = 15.0
const HEAVY_ATK_BLK_COST float32 = 20.0
const HEAVY_ATK_BLKED_DMG int = 2

//INTERRUPTABLE_STATES := map[string]bool{"standing":true,"blocking":true}
//TERMINAL_STATES := map[string]bool{"standing":true,"blocking":true,"countered":true}
var INTERRUPTABLE_STATES map[string]bool = map[string]bool{"standing": true, "blocking": true}
var TERMINAL_STATES map[string]bool = map[string]bool{"standing": true, "blocking": true, "countered": true}

func battle(player1inputChan, player2inputChan chan Message, player1updateChan, player2updateChan chan Update) {
	log.Println("in battle")
	players := []*Player{&Player{InputChan: player1inputChan, UpdateChan: player1updateChan, Command: "NONE", Life: 100, Stamina: 100, State: "standing", StateDuration: 0, Finished: ""}, &Player{InputChan: player2inputChan, UpdateChan: player2updateChan, Command: "NONE", Life: 100, Stamina: 100, State: "standing", StateDuration: 0, Finished: ""}}
	timerChan := make(chan bool)
	go clock(timerChan)
	for players[0].Life > 0 && players[1].Life > 0 {
		select {
		// Each mainloop cycle:
		case <-timerChan:
			players[0].UpdateChan <- Update{Self: players[0].Status(), Enemy: players[1].Status()}
			players[1].UpdateChan <- Update{Self: players[1].Status(), Enemy: players[0].Status()}
			for p, player := range players {
				player.PassTime(1)
				// Set the 'enemy' var to the other player, we'll need it later.
				enemy := players[1]
				if p == 1 {
					enemy = players[0]
				}
				switch player.Finished {
				case "light attack":
					if enemy.State == "blocking" {
						if enemy.Stamina >= LIGHT_ATK_BLK_COST {
							enemy.Stamina -= LIGHT_ATK_BLK_COST
							// If they haven't been blocking as long as the attack was in progress; that is, if they blocked reactively...
							if -enemy.StateDuration < LIGHT_ATK_SPD {
								// The player is counterattacked. They are placed in a stunned state that they must press a button to escape before the counterattack lands.
								player.SetState("countered", -1)
								enemy.SetState("counterattack", LIGHT_ATK_CNTR_SPD)
							}
						} else {
							// If you try to block an attack but you don't have enough stamina, you still lose your stamina and you also take damage.
							enemy.Stamina = 0.0
							enemy.Life -= LIGHT_ATK_DMG
						}
					} else {
						// If the enemy wasn't blocking, they just take damage.
						enemy.Life -= LIGHT_ATK_DMG
					}
				case "counterattack":
					// No conditions here because if you dodge the counter attack it puts the enemy out of the counterattacking state.
					enemy.Life -= LIGHT_ATK_CNTR_DMG
					enemy.SetState("standing", 0)
				case "heavy attack":
					if enemy.State == "blocking" {
						if enemy.Stamina >= HEAVY_ATK_BLK_COST {
							enemy.Stamina-=HEAVY_ATK_BLK_COST
							enemy.Life-=HEAVY_ATK_BLKED_DMG
						} else {
							enemy.Stamina=0.0
							enemy.Life-=HEAVY_ATK_DMG
						}
					} else {
							enemy.Life-=HEAVY_ATK_DMG
					}
				}
				player.Finished = ""
				switch player.Command {
				case "NONE":
					if player.State == "blocking" {
						player.SetState("standing", 0)
					}
				case "BLOCK":
					if INTERRUPTABLE_STATES[player.State] && player.State != "blocking" {
						player.SetState("blocking", 0)
					}
				case "SAVE":
					log.Println("got save command")
					if player.State == "countered" {
						log.Println("saving")
						player.SetState("standing", 0)
						enemy.SetState("standing", 0)
					}
				case "LIGHT":
					if INTERRUPTABLE_STATES[player.State] && player.Stamina >= LIGHT_ATK_COST {
						player.SetState("light attack", LIGHT_ATK_SPD)
						player.Stamina -= LIGHT_ATK_COST
					}
				case "HEAVY":
					if INTERRUPTABLE_STATES[player.State] && player.Stamina >= HEAVY_ATK_COST {
						player.SetState("heavy attack", HEAVY_ATK_SPD)
						player.Stamina -= HEAVY_ATK_COST
					}
				}
				if player.Command != "BLOCK" {
					player.Command = "NONE"
				}
			}
		case input := <-players[0].InputChan:
			if input.Content != "NONE" {
				log.Println("player0 sent:", input.Content)
			}
			players[0].Command = input.Content
		case input := <-players[1].InputChan:
			if input.Content != "NONE" {
				log.Println("player1 sent:", input.Content)
			}
			players[1].Command = input.Content
		}
	}
}

func clock(channel chan bool) {
	timer := time.NewTimer(10 * time.Millisecond)
	for range timer.C {
		timer.Reset(10 * time.Millisecond)
		channel <- true
	}
}
