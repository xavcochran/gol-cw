package stubs

var Subscribe = "Broker.Subscribe"
var Publish = "Broker.Publish"
var ProcessGol= "Broker.ProcessGol"
var CountAlive= "Broker.CountAliveCells"

var ProcessSlice = "Worker.ProcessSlice"

type Subscription struct {
	WorkerAddress string
	Function      string
}

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type StatusReport struct {
	Message string
}

type Job struct {
	World  [][]byte
	Y1, Y2 int
}

type Result struct {
	World [][]byte
}

type Request struct {
	Params Params
	World  [][]uint8
}

type Response struct {
	World       [][]uint8
	CurrentTurn int
	AliveCount int
}

type CountAliveResponse struct{
	CurrentTurn int
	AliveCount int
}

type ProcessSliceArgs struct {
	Params Params
	Y1     int
	Y2     int
	World  [][]uint8
}

type ProcessSliceResponse struct {
	World [][]uint8
}

