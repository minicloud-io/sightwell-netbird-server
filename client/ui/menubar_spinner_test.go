//go:build !android && !ios && !freebsd && !js

package main

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMainThread queues posted work so a test decides when it "runs".
type fakeMainThread struct {
	mu    sync.Mutex
	queue []func()
}

func (f *fakeMainThread) post(fn func()) {
	f.mu.Lock()
	f.queue = append(f.queue, fn)
	f.mu.Unlock()
}

func (f *fakeMainThread) drain() {
	f.mu.Lock()
	q := f.queue
	f.queue = nil
	f.mu.Unlock()
	for _, fn := range q {
		fn()
	}
}

func (f *fakeMainThread) pending() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queue)
}

func TestSpinnerFramesEmbedded(t *testing.T) {
	require.Len(t, spinnerFrames, spinnerFrameCount)
	for i, f := range spinnerFrames {
		assert.True(t, bytes.HasPrefix(f, []byte("\x89PNG")), "frame %d is a PNG", i)
	}
	assert.NotEqual(t, spinnerFrames[0], spinnerFrames[1], "frames differ")
}

func TestSpinnerAnimatesAndStops(t *testing.T) {
	main := &fakeMainThread{}
	s := &menubarSpinner{post: main.post, interval: time.Millisecond}
	var shown [][]byte
	set := func(f []byte) { shown = append(shown, f) }

	s.start(set)
	s.start(set) // idempotent: no second run
	require.Eventually(t, func() bool { return main.pending() >= 4 }, time.Second, time.Millisecond)
	main.drain()
	require.GreaterOrEqual(t, len(shown), 4)
	for i := range 4 {
		assert.Equal(t, spinnerFrames[i], shown[i], "frames advance in order from the first")
	}

	// Frames queued before stop must not land afterwards.
	require.Eventually(t, func() bool { return main.pending() >= 1 }, time.Second, time.Millisecond)
	s.stop()
	n := len(shown)
	main.drain()
	assert.Equal(t, n, len(shown), "stale frames are discarded")

	time.Sleep(10 * time.Millisecond)
	assert.Zero(t, main.pending(), "no frames after stop")
	s.stop() // idempotent
}

func TestSpinnerRestartIgnoresOldRun(t *testing.T) {
	main := &fakeMainThread{}
	s := &menubarSpinner{post: main.post, interval: time.Hour}
	var first, second int
	s.start(func([]byte) { first++ })
	s.stop()
	s.start(func([]byte) { second++ })
	main.drain()
	assert.Zero(t, first, "the stopped run's queued frame is dropped")
	assert.Equal(t, 1, second)
	s.stop()
}
