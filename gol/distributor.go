package gol

import (
	//"fmt"
	"time"
	//"uk.ac.bris.cs/gameoflife/util"
)

func distributor(p Params, c distributorChannels) {
	// Initialise world and all variables here
	world := initialiseWorld(p, c)
	turn := 0
	quit := false
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()

	c.events <- StateChange{CompletedTurns: turn, NewState: Executing}

	// Main game loop for each turn
	resultChan := make(chan [][]byte)
	// Run calculate here to initialise the first result chan
	go calculateNextState(p, c, world, turn, resultChan)

	for turn < p.Turns && !quit {
		select {
		case newFrame := <-resultChan:
			world = newFrame
			c.events <- TurnComplete{CompletedTurns: turn}
			turn++
			if turn < p.Turns {
				go calculateNextState(p, c, world, turn, resultChan)
			}
		case <-timer.C:
			reportAliveCells(c, world, turn)
		case key := <-c.keyPresses:
			handleKeyPress(key, c, &world, p, &turn, &quit)
		}
	}

	finaliseGame(c, world, p, turn)
}
