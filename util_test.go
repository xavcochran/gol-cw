package main

import (
	"fmt"
	"testing"
	"time"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

type Tester struct {
	t            *testing.T
	params       gol.Params
	keyPresses   chan<- rune
	events       <-chan gol.Event
	eventWatcher chan gol.Event
	quitting     chan bool
	golDone      <-chan bool
	turn         int
	world        [][]byte
	aliveMap     map[int]int
	sdlSync      chan bool
}

func MakeTester(
	t *testing.T,
	params gol.Params,
	keyPresses chan<- rune,
	events <-chan gol.Event,
	golDone <-chan bool,
) Tester {
	world := make([][]byte, params.ImageHeight)
	for i := range world {
		world[i] = make([]byte, params.ImageWidth)
	}

	W.ClearPixels()
	emptyOutFolder()

	eventWatcher := make(chan gol.Event, 1000)
	return Tester{
		t:            t,
		params:       params,
		keyPresses:   keyPresses,
		events:       events,
		eventWatcher: eventWatcher,
		quitting:     make(chan bool),
		golDone:      golDone,
		turn:         0,
		world:        world,
		aliveMap:     readAliveCounts(params.ImageWidth, params.ImageHeight),
		sdlSync:      nil,
	}
}

func (tester *Tester) SetTestSdl() {
	tester.sdlSync = make(chan bool)
}

func (tester *Tester) Loop() {

	avgTurns := util.NewAvgTurns()

	for {
		select {
		case quitPanic := <-tester.quitting:
			awaitDone := func() {
				for {
					select {
					case <-tester.golDone:
						return
					case <-tester.events:
					}
				}

			}

			if quitPanic {
				timeout(2*time.Second, awaitDone, "Your program has not returned from the gol.Run function")
			} else {
				timeoutWarn(2*time.Second, awaitDone, "Your program has not returned from the gol.Run function\n%v\n%v", "Continuing with other tests", "You may get unexpected behaviour")
			}

			// cancelDeadline <- true
			return
		case event := <-tester.events:
			switch e := event.(type) {
			case gol.CellFlipped:
				if tester.sdlSync != nil {
					assert(e.CompletedTurns == tester.turn,
						"Expected completed %v turns, got %v instead", tester.turn, e.CompletedTurns)
				}
				tester.world[e.Cell.Y][e.Cell.X] = ^tester.world[e.Cell.Y][e.Cell.X]
				if W != nil {
					W.FlipPixel(e.Cell.X, e.Cell.Y)
				}
			case gol.CellsFlipped:
				if tester.sdlSync != nil {
					assert(e.CompletedTurns == tester.turn,
						"Expected completed %v turns, got %v instead", tester.turn, e.CompletedTurns)
				}
				for _, cell := range e.Cells {
					tester.world[cell.Y][cell.X] = ^tester.world[cell.Y][cell.X]
					if W != nil {
						W.FlipPixel(cell.X, cell.Y)
					}
				}
			case gol.TurnComplete:

				tester.turn++
				if tester.sdlSync != nil {
					assert(e.CompletedTurns == tester.turn,
						"Expected completed %v turns, got %v instead", tester.turn, e.CompletedTurns)
				}

				if Refresh != nil {
					Refresh <- true
				}

				if tester.sdlSync != nil {
					// tester.testAlive()
					// tester.testImage()
					tester.sdlSync <- true
					<-tester.sdlSync
				}
			case gol.AliveCellsCount:
				fmt.Printf("Completed Turns %-8v %-20v Avg%+5v turns/sec\n", event.GetCompletedTurns(), event, avgTurns.Get(event.GetCompletedTurns()))
			case gol.ImageOutputComplete:
				fmt.Printf("Completed Turns %-8v %v\n", event.GetCompletedTurns(), event)
				tester.eventWatcher <- e
			case gol.FinalTurnComplete:
				fmt.Printf("Completed Turns %-8v %v\n", event.GetCompletedTurns(), event)
				tester.eventWatcher <- e
			case gol.StateChange:
				fmt.Printf("Completed Turns %-8v %v\n", event.GetCompletedTurns(), event)
				tester.eventWatcher <- e

				if tester.sdlSync != nil && tester.turn == 0 {
					// tester.testAlive()
					// tester.testImage()
					tester.sdlSync <- true
					<-tester.sdlSync
				}
			}
		}
	}
}

func (tester *Tester) Stop(returnPanic bool) {
	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-tester.sdlSync:
				tester.sdlSync <- true
			case <-stop:
				return
			}
		}
	}()

	tester.quitting <- returnPanic
	stop <- true
}

func (tester *Tester) AwaitTurn() int {
	timeout(2*time.Second, func() {
		<-tester.sdlSync
	}, "No turns completed in 2 seconds. Is your program deadlocked?")
	return tester.turn
}

func (tester *Tester) Continue() {
	tester.sdlSync <- true
}

func (tester *Tester) TestAlive() {
	aliveCount := 0
	for _, row := range tester.world {
		for _, cell := range row {
			if cell == 0xFF {
				aliveCount++
			}
		}
	}
	expected := 0

	if tester.turn <= 10000 {
		expected = tester.aliveMap[tester.turn]
	} else if tester.turn%2 == 0 {
		expected = 5565
	} else {
		expected = 5567
	}
	assert(aliveCount == expected,
		"At turn %v expected %v alive cells in the SDL window, got %v instead", tester.turn, expected, aliveCount)
}

func (tester *Tester) TestImage() {
	if tester.turn == 0 || tester.turn == 1 || tester.turn == 100 {
		tester.t.Logf("Checking SDL image at turn %v", tester.turn)

		width, height := tester.params.ImageWidth, tester.params.ImageHeight

		path := fmt.Sprintf("check/images/%vx%vx%v.pgm", width, height, tester.turn)
		expectedAlive := readAliveCells(path, width, height)

		aliveCells := make([]util.Cell, 0, width*height)
		for y := range tester.world {
			for x, cell := range tester.world[y] {
				if cell == 255 {
					aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
				}
			}
		}

		equal := checkEqualBoard(aliveCells, expectedAlive)
		if !equal {
			if tester.turn == 0 {
				tester.t.Error("The image displayed in the SDL window is incorrect for turn 0\nHave you sent the correct CellFlipped events before StateChange Executing?")
			} else {
				tester.t.Errorf("The image displayed in the SDL window is incorrect for turn %v", tester.turn)
			}
		}
	} else {
		fmt.Printf("WARNING: TestImage called on invalid turn: %v. This call will be ignored\n", tester.turn)
	}
}

func (tester *Tester) TestStartsExecuting() {
	tester.t.Logf("Testing for first StateChange Executing event")
	timeout(2*time.Second, func() {
		e := <-tester.eventWatcher
		if e, ok := e.(gol.StateChange); ok {
			assert(e.NewState == gol.Executing,
				"First StateChange event should have a NewState of Executing, not %v", e)
			assert(e.CompletedTurns == 0,
				"First StateChange event should have a CompletedTurns of 0, not %v", e.CompletedTurns)
			return
		}

		panic(fmt.Sprintf("%v event should not be sent before StateChange Executing", e))

	}, "No StateChange events received in 2 seconds")
}

func (tester *Tester) TestExecutes() {
	tester.t.Logf("Testing for StateChange Executing event")
	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.StateChange); ok && e.NewState == gol.Executing {
				return
			}
		}
	}, "No StateChange Executing events received in 2 seconds")
}

func (tester *Tester) TestPauses() {
	tester.t.Logf("Testing for StateChange Paused event")
	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.StateChange); ok && e.NewState == gol.Paused {
				return
			}
		}
	}, "No StateChange Paused events received in 2 seconds")
}

func (tester *Tester) TestQuits() {
	tester.t.Logf("Testing for StateChange Quitting event")
	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.StateChange); ok && e.NewState == gol.Quitting {
				return
			}
		}
	}, "No StateChange Quitting events received in 2 seconds")
}

func (tester *Tester) TestNoStateChange(ddl time.Duration) {
	change := make(chan gol.StateChange, 1)
	stop := make(chan bool)
	go func() {
		for {
			select {
			case e := <-tester.eventWatcher:
				if e, ok := e.(gol.StateChange); ok {
					change <- e
					return
				}
			case <-stop:
				return
			}
		}
	}()
	select {
	case <-time.After(ddl):
		stop <- true
		return
	case e := <-change:
		panic(fmt.Sprintf("Recieved unexpected StateChange event %v", e))
	}
}

func (tester *Tester) TestOutput() {
	width, height := tester.params.ImageWidth, tester.params.ImageHeight
	tester.t.Logf("Testing image output")

	turn := make(chan int, 1)

	timeout(4*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.ImageOutputComplete); ok {
				assert(e.Filename == fmt.Sprintf("%vx%vx%v", width, height, e.CompletedTurns),
					"Filename is not correct")
				turn <- e.CompletedTurns
				return
			}
		}
	}, "No ImageOutput events received in 4 seconds\n%v",
		"If this test is running in WSL2, please make sure the test is located within WSL2 file system rather than Windows! i.e. Your path must not start with /mnt/...")

	eventTurn := <-turn

	expected := 0
	if eventTurn <= 10000 {
		expected = tester.aliveMap[eventTurn]
	} else if eventTurn%2 == 0 {
		expected = 5565
	} else {
		expected = 5567
	}

	path := fmt.Sprintf("out/%vx%vx%v.pgm", width, height, eventTurn)

	// time.Sleep(1 * time.Second)

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Failed to read image file. Make sure you do ioCheckIdle before sending the ImageOutputComplete\n%v", r))
		}
	}()
	alive := readAliveCells(path, width, height)

	assert(len(alive) == expected, "At turn %v expected %v alive cells in output PGM image, got %v instead", eventTurn, expected, len(alive))
}

func (tester *Tester) TestPause(delay time.Duration) {
	// time.Sleep(delay)
	tester.t.Logf("Testing Pause key pressed")
	// for len(tester.eventWatcher) > 0 {
	// 	<-tester.eventWatcher
	// }
	tester.keyPresses <- 'p'
	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.StateChange); ok && e.NewState == gol.Paused {
				return
			}
		}
	}, "No Pause events received in 2 seconds")

	// tester.TestOutput(2 * time.Second)

	time.Sleep(2 * time.Second)
	tester.t.Logf("Testing Pause key pressed again")
	tester.keyPresses <- 'p'
	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.StateChange); ok && e.NewState == gol.Executing {
				return
			}
		}
	}, "No Executing events received in 2 seconds")
}

func (tester *Tester) TestQuitting(delay time.Duration) {
	time.Sleep(delay)
	tester.t.Logf("Testing Quit key pressed")
	for len(tester.eventWatcher) > 0 {
		<-tester.eventWatcher
	}
	tester.keyPresses <- 'q'
	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if _, ok := e.(gol.FinalTurnComplete); ok {
				return
			}
		}
	}, "No FinalTurnComplete events received in 2 seconds")

	timeout(4*time.Second, func() {
		for e := range tester.eventWatcher {
			if _, ok := e.(gol.ImageOutputComplete); ok {
				return
			}
		}
	}, "No ImageOutput events received in 4 seconds")

	timeout(2*time.Second, func() {
		for e := range tester.eventWatcher {
			if e, ok := e.(gol.StateChange); ok && e.NewState == gol.Quitting {
				return
			}
		}
	}, "No Quitting events received in 2 seconds")
}

func deadline(ddl time.Duration, msg string) chan<- bool {
	done := make(chan bool, 1)
	go func() {
		select {
		case <-time.After(ddl):
			panic(msg)
		case <-done:
			return
		}
	}()
	return done
}

func timeout(ddl time.Duration, f func(), msg string, a ...interface{}) {
	done := make(chan bool, 1)
	go func() {
		f()
		done <- true
	}()
	select {
	case <-time.After(ddl):
		panic(fmt.Sprintf(msg, a...))
	case <-done:
		return
	}
}

func timeoutWarn(ddl time.Duration, f func(), msg string, a ...interface{}) {
	done := make(chan bool, 1)
	go func() {
		f()
		done <- true
	}()
	select {
	case <-time.After(ddl):
		fmt.Printf("WARNING: %v\n", fmt.Sprintf(msg, a...))
	case <-done:
		return
	}
}

func assert(predicate bool, msg string, a ...interface{}) {
	if !predicate {
		panic(fmt.Sprintf(msg, a...))
	}
}
