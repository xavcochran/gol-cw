package gol

import (
	"fmt"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	turn := 0

	c.events <- StateChange{turn, Executing}
	timer := time.NewTicker(2 * time.Second)

	// 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	filename := fmt.Sprintf("%vx%v", p.ImageWidth, p.ImageHeight)
	fmt.Println(filename)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	// writes image
	outFilename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)

	// Read each cell into world
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			cell := <-c.ioInput
			if cell == 255 {
				c.events <- CellFlipped{CompletedTurns: 0, Cell: util.Cell{X: x, Y: y}}
			}
			world[y][x] = cell
		}
	}

	resultChan := make(chan [][]uint8)

	// Execute all turns of the Game of Life.
	go calculateNextState(p, c, world, turn, resultChan)
	for turn < p.Turns {
		fmt.Println("TURN", turn, p.Turns)
		select {
		case newFrame := <-resultChan:
			world = newFrame
			turn++
			if turn < p.Turns {
				go calculateNextState(p, c, world, turn, resultChan)
			}
		case <-timer.C:
			aliveCells := calculateAliveCells(world)                                       // calculates alive cells and stores it in aliveCells
			c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: len(aliveCells)} // sends event
		}

	}

	writeImage(c, outFilename, world, p, turn)

	// Report the final state using FinalTurnCompleteEvent.
	liveCells := calculateAliveCells(world)
	c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: liveCells}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
