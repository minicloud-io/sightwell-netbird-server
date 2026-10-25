//go:build !android && !ios && !freebsd && !js

package main

import (
	"embed"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Sightwell fork: while the client is connecting, the macOS menu bar shows
// the dashboard's spinner (the mark inside a track with a turning orange arc)
// instead of upstream's template icon with a dots badge. The frames come from
// build/sightwell/icons.py.

//go:embed assets/sightwell-menubar-connecting-*.png
var spinnerFS embed.FS

const (
	spinnerFrameCount = 12
	// One full turn per second, like the dashboard spinner.
	spinnerInterval = time.Second / spinnerFrameCount
)

var spinnerFrames = loadSpinnerFrames()

func loadSpinnerFrames() [][]byte {
	frames := make([][]byte, spinnerFrameCount)
	for i := range frames {
		b, err := spinnerFS.ReadFile(fmt.Sprintf("assets/sightwell-menubar-connecting-%02d.png", i))
		if err != nil {
			panic(err) // the embed pattern guarantees the files; a gap is a build bug
		}
		frames[i] = b
	}
	return frames
}

// menubarSpinner cycles the spinner frames until stopped. Its zero value is
// ready to use.
//
// Frames are posted to the main thread asynchronously and each one checks,
// when it runs there, that its run is still the current one. stop bumps the
// generation before the caller sets the next icon, and the main thread runs
// work in order, so a frame queued before the state change can never land
// after the new icon. Nothing blocks on the main thread, so start and stop are
// safe to call from it (NewTray) and from any goroutine.
type menubarSpinner struct {
	mu   sync.Mutex
	quit chan struct{} // non-nil while a run is active; guarded by mu
	gen  atomic.Uint64

	// Test hooks; nil means the Wails main thread and spinnerInterval.
	post     func(func())
	interval time.Duration
}

// start begins animating with set, or does nothing if already running. set
// runs on the main thread.
func (s *menubarSpinner) start(set func(frame []byte)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quit != nil {
		return
	}
	quit := make(chan struct{})
	s.quit = quit
	run := s.gen.Add(1)
	post := s.post
	if post == nil {
		post = application.InvokeAsync
	}
	interval := s.interval
	if interval == 0 {
		interval = spinnerInterval
	}
	show := func(i int) {
		frame := spinnerFrames[i%len(spinnerFrames)]
		post(func() {
			if s.gen.Load() == run {
				set(frame)
			}
		})
	}
	show(0)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for i := 1; ; i++ {
			select {
			case <-quit:
				return
			case <-ticker.C:
				show(i)
			}
		}
	}()
}

// stop ends the current run, if any. Frames already queued are discarded.
func (s *menubarSpinner) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quit == nil {
		return
	}
	close(s.quit)
	s.quit = nil
	s.gen.Add(1)
}
