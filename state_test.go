package main

import (
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

// TestState tests for StateChange Executing and Quitting events
func TestState(t *testing.T) {
	params := gol.Params{
		Turns:       100,
		Threads:     8,
		ImageWidth:  512,
		ImageHeight: 512,
	}

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	golDone := make(chan bool, 1)
	go func() {
		gol.Run(params, events, keyPresses)
		golDone <- true
	}()

	tester := MakeTester(t, params, keyPresses, events, golDone)

	go func() {
		tester.TestStartsExecuting()
		tester.TestQuits()
		tester.Stop(true)
	}()

	tester.Loop()
}
