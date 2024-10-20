package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

// sends event when cell value changes to notify gui
func setCell(y, x int, world func(y, x int) uint8, newValue uint8, events chan<- Event, turn int) {
	if world(y, x) != newValue {
		events <- CellFlipped{CompletedTurns: turn, Cell: util.Cell{X: x, Y: y}}
	}
}

// takes y bounds for given slice of the image and calculates the next state of the world for that slice
func worker(p Params, y1, y2 int, world func(y, x int) uint8, res chan<- [][]uint8, turn int, events chan<- Event) {
	slice := (y2 - y1) + 1

	newWorld := make([][]byte, slice)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}

	// go through each row and column and calculate if the cell should be alive or dead
	for i := 0; i < p.ImageWidth; i++ {
		for j := y1; j <= y2; j++ {
			aliveNeighbors := 0
			// check the 8 neighbors of the cell
			for x := -1; x <= 1; x++ {
				for y := -1; y <= 1; y++ {
					if x == 0 && y == 0 {
						continue
					}
					// check if the neighbor is alive
					x_neighbour := (i + x + p.ImageWidth) % p.ImageWidth
					y_neighbour := (j + y + p.ImageHeight) % p.ImageHeight

					if world(y_neighbour, x_neighbour) == 255 {
						aliveNeighbors++
					}
				}
			}
			if world(j, i) == 255 {
				if aliveNeighbors < 2 || aliveNeighbors > 3 {
					setCell(j, i, world, 0, events, turn)
					newWorld[j-y1][i] = 0
				} else {
					setCell(j, i, world, 255, events, turn)
					newWorld[j-y1][i] = 255
				}
			} else {
				if aliveNeighbors == 3 {
					setCell(j, i, world, 255, events, turn)
					newWorld[j-y1][i] = 255
				} else {
					setCell(j, i, world, 0, events, turn)
					newWorld[j-y1][i] = 0
				}
			}
		}
	}
	res <- newWorld
}
