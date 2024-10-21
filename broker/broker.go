package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"

)

var jobs = make(chan Jobs, 16)

type Jobs struct {
	Args       stubs.ProcessSliceArgs
	ReturnChan chan [][]uint8
}

// runs worker 
// worker waits for jobs to be posted to jobs channel and then consumes them from the channel and sends the job to the worker
func runJobs(client *rpc.Client) {
	for {
		job := <-jobs
		err := workerReq(job.Args, client, job.ReturnChan)
		if err != nil {
			fmt.Println("Error with request")
			jobs <- job
		}
	}
}

func calculateAliveCells(world [][]byte) int{
	var aliveCells  int
	for _, row := range world {
		for _, cellValue := range row {
			if cellValue == 255 {
				aliveCells++
			}
		}
	}
	return aliveCells
}

func calculateNextState(p stubs.Params, b *Broker) [][]uint8 {

	// new empty 2d world
	newWorld := make([][]byte, 0)

	// makes channel for each worker
	resChans := make([]chan [][]uint8, p.Threads)
	for i := range resChans {
		resChans[i] = make(chan [][]uint8)
	}

	// immutableWorld := makeImmutableWorld(b.world)

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

		//publish
		if len(b.world) == 16 {
			fmt.Println(b.world)
		}
		newJob := Jobs{
			Args:       stubs.ProcessSliceArgs{Params: p, Y1: y1, Y2: y2, World: b.world},
			ReturnChan: resChan,
		}
		jobs <- newJob

	}

	for _, resChan := range resChans {
		// gets slice from worker and appends to frame
		// blocks on channels sequentially so no need for wait groups or additional processing for ordering
		newWorld = append(newWorld, <-resChan...)
	}
	b.wg.Done()

	return newWorld
}

// sends job request to worker
func workerReq(args stubs.ProcessSliceArgs, client *rpc.Client, responseChan chan<- [][]uint8) error {

	response := &stubs.ProcessSliceResponse{}
	err := client.Call(stubs.ProcessSlice, args, response)
	if err != nil {
		return err
	}
	responseChan <- response.World
	return nil
}

type Broker struct {
	world   [][]uint8
	turn    int
	lock    sync.Mutex
	pause   bool
	quit    bool
	signal  chan string
	wg      sync.WaitGroup
}

// subscribes the worker to the broker so that the broker can send work to the worker
func (b *Broker) Subscribe(req stubs.Subscription, res *stubs.StatusReport) (err error) {
	// dials worker
	client, err := rpc.Dial("tcp", req.WorkerAddress)
	if err != nil {
		fmt.Println("Error subscribing ", req.WorkerAddress)
		fmt.Println(err)
		return err
	}
	fmt.Println("Connected", req.WorkerAddress)

	// runs worker 
	go runJobs(client)
	return err
}

// iterates through the number of turns and calculates the next state for each turn
func (b *Broker) ProcessGol(req stubs.Request, res *stubs.Response) (err error) {
	b.world = req.World
	for b.turn = 0; b.turn  < req.Params.Turns; b.turn++ {
		b.wg.Add(1)

		newWorld := calculateNextState(req.Params, b)
		b.world = newWorld
	}
	b.wg.Wait()
	res.World = b.world
	res.CurrentTurn = req.Params.Turns
	return nil
}

func (b *Broker) CountAliveCells(req stubs.Request, res *stubs.CountAliveResponse) (err error) {
	b.wg.Add(1)
	defer b.wg.Done()

	b.lock.Lock()
	count := calculateAliveCells(b.world)
	res.CurrentTurn = b.turn
	res.AliveCount = count
	b.lock.Unlock()

	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	listener, e := net.Listen("tcp", ":"+*pAddr)
	if e != nil {
		fmt.Println(e)
	}
	defer listener.Close()
	rpc.Accept(listener)
}
