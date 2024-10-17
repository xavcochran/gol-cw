package gol

import (
	"fmt"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// closure to return return immutable cell to avoid overwriting
func makeImmutableWorld(world [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return world[y][x]
	}
}

// standard calculate alive cells function from labs
func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var aliveCells []util.Cell
	for y, row := range world {
		for x, cellValue := range row {
			if cellValue == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

func calculateNextState(p Params, c distributorChannels, world [][]uint8, turn int, res chan<- [][]uint8) {

	// new empty 2d world
	newWorld := make([][]byte, 0)

	// makes channel for each worker
	resChans := make([]chan [][]uint8, p.Threads)
	for i := range resChans {
		resChans[i] = make(chan [][]uint8)
	}

	immutableWorld := makeImmutableWorld(world)

	// divides up world between threads
	vDiff := p.ImageHeight / p.Threads
	// handles remainder e.g. if num of threads is odd
	remainder := p.ImageHeight % p.Threads

	for i, resChan := range resChans {
		i += 1
		// Calculate y bounds for thread
		y1 := (i - 1) * vDiff
		y2 := (i * vDiff) - 1
		if i == p.Threads {
			y2 += remainder
		}

		go worker(p, y1, y2, immutableWorld, resChan, turn, c.events)
	}
	fmt.Println("done")
	for _, resChan := range resChans {
		// gets slice from worker and appends to frame
		// blocks on channels sequentially so no need for wait groups or additional processing for ordering
		newSlice := <-resChan
		newWorld = append(newWorld, newSlice...)
	}

	res <- newWorld
}

func writeImage(c distributorChannels, filename string, world [][]uint8, p Params, turn int) {
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	// writes image to event channel byte by byte
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	// send image output completed event to user
	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	turn := 0

	c.events <- StateChange{turn, Executing}
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
	// TODO: Execute all turns of the Game of Life.

	go calculateNextState(p, c, world, turn, resultChan)
	for turn < p.Turns {
		fmt.Println("TURN", turn, p.Turns)
		newFrame := <-resultChan
		world = newFrame
		turn++
		if turn < p.Turns {
			go calculateNextState(p, c, world, turn, resultChan)
		}

	}

	writeImage(c, outFilename, world, p, turn)

	// TODO: Report the final state using FinalTurnCompleteEvent.
	liveCells := calculateAliveCells(p, world)
	c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: liveCells}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
