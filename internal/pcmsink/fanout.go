// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package pcmsink

import (
	"sync"
	"sync/atomic"
	"time"
)

// sinkQueueBytes bounds per-sink buffering to a little over a second of
// audio. Past that point chunks are dropped so a stalled sink cannot block
// the others or grow memory without bound.
const sinkQueueBytes = 256 << 10

const sinkQueueChunks = sinkQueueBytes / chunkBytes

// SinkStatus reports a sink spec, its last error, and dropped chunk count.
type SinkStatus struct {
	Spec    string `json:"spec"`
	Error   string `json:"error,omitempty"`
	Dropped uint64 `json:"dropped,omitempty"`
}

// Fanout multicasts PCM chunks to every registered sink. Each sink gets its
// own buffered queue and writer goroutine, so Write never blocks on a slow
// consumer.
type Fanout struct {
	mu      sync.Mutex
	workers map[string]*sinkWorker
	closed  bool
}

type sinkWorker struct {
	sink    Sink
	queue   chan []byte
	done    chan struct{}
	dropped atomic.Uint64
	errMsg  atomic.Value
	wg      sync.WaitGroup
}

// NewFanout builds the sinks for the given target specs. Individual sink
// failures are collected and returned as a combined error while the sinks
// that did open are still registered.
func NewFanout(specs []string) (*Fanout, error) {
	f := &Fanout{workers: make(map[string]*sinkWorker)}
	var firstErr error
	for _, spec := range specs {
		sink, err := New(spec)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		f.add(spec, sink)
	}
	return f, firstErr
}

func (f *Fanout) add(name string, sink Sink) {
	w := &sinkWorker{
		sink:  sink,
		queue: make(chan []byte, sinkQueueChunks),
		done:  make(chan struct{}),
	}
	f.workers[name] = w
	w.wg.Add(1)
	go w.run()
}

// Write broadcasts one PCM chunk to all sinks. Chunks are copied so callers
// may reuse their buffer.
func (f *Fanout) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return
	}
	buf := append([]byte(nil), p...)
	for _, w := range f.workers {
		select {
		case w.queue <- buf:
		default:
			w.dropped.Add(1)
		}
	}
}

func (w *sinkWorker) run() {
	defer w.wg.Done()
	for {
		select {
		case p := <-w.queue:
			if _, err := w.sink.Write(p); err != nil {
				w.errMsg.Store(err.Error())
			}
		case <-w.done:
			return
		}
	}
}

// Status returns per-sink health for diagnostics.
func (f *Fanout) Status() []SinkStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]SinkStatus, 0, len(f.workers))
	for name, w := range f.workers {
		st := SinkStatus{Spec: name, Dropped: w.dropped.Load()}
		if v, ok := w.errMsg.Load().(string); ok {
			st.Error = v
		} else if reporter, ok := w.sink.(interface{ statusErr() string }); ok {
			st.Error = reporter.statusErr()
		}
		out = append(out, st)
	}
	return out
}

// Close shuts down all sinks and their writer goroutines.
func (f *Fanout) Close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	workers := f.workers
	f.workers = nil
	f.mu.Unlock()

	for _, w := range workers {
		close(w.done)
	}
	for _, w := range workers {
		waitCh := make(chan struct{})
		go func(w *sinkWorker) {
			w.wg.Wait()
			close(waitCh)
		}(w)
		select {
		case <-waitCh:
		case <-time.After(2 * time.Second):
			// A sink blocked in Write must not wedge shutdown.
		}
		_ = w.sink.Close()
	}
}
