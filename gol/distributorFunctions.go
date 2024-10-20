package gol

import (
	//"fmt"

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
	keyPresses <-chan rune
}

// closure to return return immutable cell to avoid overwriting
func makeImmutableWorld(world [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return world[y][x]
	}
}

// standard calculate alive cells function from labs
func calculateAliveCells(world [][]byte) []util.Cell {
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

// calculates the next state of all the cells
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
	for _, resChan := range resChans {
		// gets slice from worker and appends to frame
		// blocks on channels sequentially so no need for wait groups or additional processing for ordering
		newSlice := <-resChan
		newWorld = append(newWorld, newSlice...)
	}

	res <- newWorld
}

// writes the image to a pgm file
func writeImage(c distributorChannels, world [][]uint8, p Params, turn int) {
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turn)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	// writes image to event channel byte by byte
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	// send image output completed event to user
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	fmt.Println("sending image", filename)
	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}

// initialises the world and sets it to the state stored in the image
func initialiseWorld(p Params, c distributorChannels) [][]byte {
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			cell := <-c.ioInput
			if cell == 255 {
				c.events <- CellFlipped{CompletedTurns: 0, Cell: util.Cell{X: x, Y: y}}
			}
			world[y][x] = cell
		}
	}

	return world
}

// calculates the number of allive cells in the world
func reportAliveCells(c distributorChannels, world [][]byte, turn int) {
	aliveCells := calculateAliveCells(world)
	c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: len(aliveCells)}
}

// clean up when game is over
func finaliseGame(c distributorChannels, world [][]byte, p Params, turn int) {
	writeImage(c, world, p, turn)
	liveCells := calculateAliveCells(world)
	c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: liveCells}

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{CompletedTurns: turn, NewState: Quitting}
	close(c.events)
}

// quit function when user presses q
func handleQuit(c distributorChannels, turn int, quit *bool) {
	*quit = true
	c.events <- StateChange{CompletedTurns: turn, NewState: Quitting}
}

// handles the keypress the user makes while GOL is paused
func handlePausedState(c distributorChannels, world *[][]byte, p Params, turn *int, quit *bool) {
	for {
		key := <-c.keyPresses
		switch key {
		case 'p':
			c.events <- StateChange{CompletedTurns: *turn, NewState: Executing}
			return
		case 'q':
			handleQuit(c, *turn, quit)
			return
		case 's':
			writeImage(c, *world, p, *turn)
		}
	}
}

// handles the users keypress
func handleKeyPress(key rune, c distributorChannels, world *[][]byte, p Params, turn *int, quit *bool) {
	switch key {
	case 'p':
		c.events <- StateChange{CompletedTurns: *turn, NewState: Paused}
		handlePausedState(c, world, p, turn, quit)
	case 'q':
		handleQuit(c, *turn, quit)
	case 's':
		writeImage(c, *world, p, *turn)
	}
}
