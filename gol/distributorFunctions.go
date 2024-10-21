package gol

import (
	"fmt"

	"net/rpc"

	// "uk.ac.bris.cs/gameoflife/stubs"
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



func writeImage(c distributorChannels, world [][]uint8, p Params, turn int) {
	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, turn)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	// Writes image to event channel byte by byte
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	// Send image output completed event to user
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	fmt.Println("sending image", filename)
	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: filename}
}


func connectToBroker() (*rpc.Client, error) {
	//Dial broker address.
	client, err := rpc.Dial("tcp",  "127.0.0.1:8030")
	

	return client, err
}

// Calculates the number of alive cells in the world
func reportAliveCells(c distributorChannels, world [][]byte, turn int) {
	aliveCells := calculateAliveCells(world)
	c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: len(aliveCells)}
}