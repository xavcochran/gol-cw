package gol

import (
	"fmt"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	// "uk.ac.bris.cs/gameoflife/util"
)

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	client, err := connectToBroker()
	if err != nil {
		fmt.Println("error connecting to broker", err)
		client.Close()
		return
	}
	defer client.Close()
	// c.events <- StateChange{0, Executing}
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
	// outFilename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)

	// Read each cell into world
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			cell := <-c.ioInput
			world[y][x] = cell
		}
	}

	resChan := make(chan stubs.Response)


	go func() {
		request := stubs.Request{Params: stubs.Params(p), World: world}
		response := &stubs.Response{}

		client.Call(stubs.ProcessGol, request, response)
		resChan<-*response
	}()


	shouldQuit := false
	outerLoop:
	for !shouldQuit {
		select{
		case res := <-resChan:

			c.events <- FinalTurnComplete{CompletedTurns: res.CurrentTurn, Alive: calculateAliveCells(res.World)}
			c.events <- StateChange{res.CurrentTurn, Quitting}
			writeImage(c, res.World, p, res.CurrentTurn)
			fmt.Println("completed")
			break outerLoop
		case <-timer.C:
			reportAliveCells(c,p, world, client)
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
