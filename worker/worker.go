package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"
)

var Gol = "Worker.ProcessSlice"

type Worker struct{}

// takes y bounds for given slice of the image and calculates the next state of the world for that slice
func (w *Worker) ProcessSlice(args stubs.ProcessSliceArgs, reply *stubs.ProcessSliceResponse) error {
	fmt.Println("WORKING")
    p := args.Params
    y1 := args.Y1
    y2 := args.Y2
    world := args.World
    // turn := args.Turn

	slice := (y2 - y1) + 1

	newWorld := make([][]byte, slice)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}
	if len(world) < 17 {
		fmt.Println(world)
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

					if world[y_neighbour][x_neighbour] == 255 {
						aliveNeighbors++
					}
				}
			}
			if world[j][i] == 255 {
				if aliveNeighbors < 2 || aliveNeighbors > 3 {
					newWorld[j-y1][i] = 0
				} else {
					newWorld[j-y1][i] = 255
				}
			} else {
				if aliveNeighbors == 3 {
					newWorld[j-y1][i] = 255
				} else {
					newWorld[j-y1][i] = 0
				}
			}
		}
	}
	reply.World = newWorld
	return nil
}


func main() {
	pAddr := flag.String("ip", "127.0.0.1:8080", "IP and port to listen on")
	brokerAddr := flag.String("broker", "127.0.0.1:8030", "Address of broker instance")

	// registers methods
	err := rpc.Register(&Worker{})
	if err != nil {
		fmt.Println("Error registering Factory:", err)
		return
	}

	// dials broker
	rpcB, err := rpc.Dial("tcp", *brokerAddr)
	if err != nil {
		fmt.Println("Error dialing broker:", err)
		return
	}
	defer rpcB.Close()

	// creates subscription request
	subscription := stubs.Subscription{
		WorkerAddress: *pAddr,
		Function: Gol,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	// runs listener to recieve subscription request from broker
	go func(){
		defer wg.Done()
		ln, err := net.Listen("tcp", *pAddr)
		if err != nil {
			fmt.Println("Error creating listener on port", *pAddr, ":", err)
			return
		}
		defer ln.Close()

		fmt.Println("Factory server listening on", *pAddr)

		// Accept incoming connections
		rpc.Accept(ln)
	}()

	// subscribes to broker to recieve work
	var response stubs.StatusReport 
	err = rpcB.Call("Broker.Subscribe", subscription, &response)
	if err != nil {
		fmt.Println("Error calling Broker.Subscribe:", err)
		return
	}

	wg.Wait()
}