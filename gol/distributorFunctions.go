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
