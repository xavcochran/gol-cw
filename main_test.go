package main

import (
	"flag"
	"os"
	"runtime"
	"testing"
	"time"

	"uk.ac.bris.cs/gameoflife/sdl"
)

var W *sdl.Window
var Refresh chan bool

func TestMain(m *testing.M) {
	runtime.LockOSThread()
	var sdlFlag = flag.Bool(
		"sdl",
		false,
		"Enable the SDL window for testing.")

	flag.Parse()
	done := make(chan int, 1)
	test := func() { done <- m.Run() }
	if !(*sdlFlag) {
		go test()
	} else {
		W = sdl.NewWindow(512, 512)
		Refresh = make(chan bool, 1)
		fps := 60
		ticker := time.NewTicker(time.Second / time.Duration(fps))
		dirty := false
		go test()
	loop:
		for {
			select {
			case code := <-done:
				done <- code
				W.Destroy()
				break loop
			case <-ticker.C:
				W.PollEvent()
				if dirty {
					W.RenderFrame()
					dirty = false
				}
			case <-Refresh:
				dirty = true
			}
		}
	}
	os.Exit(<-done)
}
