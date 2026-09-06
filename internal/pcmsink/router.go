// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package pcmsink

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Source is one PCM producer: a named pipe the audio backend writes to and
// the router reads from. Each player instance gets its own source so
// concurrent players (gapless handoff, crossfade) never interleave bytes in
// a shared pipe.
type Source struct {
	path   string
	file   *os.File
	notify chan struct{}

	mu   sync.Mutex
	buf  []byte
	dead bool
}

// Path returns the FIFO path to hand to the audio backend.
func (s *Source) Path() string { return s.path }

func (s *Source) push(chunk []byte) {
	s.mu.Lock()
	if len(s.buf)+len(chunk) > sourceBufCap {
		s.mu.Unlock()
		return
	}
	s.buf = append(s.buf, chunk...)
	s.mu.Unlock()
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

// drain pops up to max bytes, rounded down to a whole s16 stereo frame.
func (s *Source) drain(max int) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := min(len(s.buf), max)
	n -= n % frameBytes
	if n <= 0 {
		return nil
	}
	out := append([]byte(nil), s.buf[:n]...)
	s.buf = s.buf[n:]
	return out
}

// Router reads PCM from per-player FIFOs, mixes concurrent streams, and
// feeds a Fanout. mpv paces its pipe writes to realtime, so the mixed output
// stays paced without an extra clock.
type Router struct {
	dir    string
	fanout *Fanout

	mu      sync.Mutex
	sources map[*Source]struct{}
	seq     int
	closed  bool

	dataCh chan *Source
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewRouter creates dir for FIFOs and starts the mixing loop.
func NewRouter(dir string, fanout *Fanout) (*Router, error) {
	if fanout == nil {
		return nil, errors.New("pcmsink: router needs a fanout")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	r := &Router{
		dir:     dir,
		fanout:  fanout,
		sources: make(map[*Source]struct{}),
		dataCh:  make(chan *Source, 64),
		done:    make(chan struct{}),
	}
	r.wg.Add(1)
	go r.mixLoop()
	return r, nil
}

// Fanout returns the sink set the router writes to.
func (r *Router) Fanout() *Fanout { return r.fanout }

// NewSource creates a FIFO and starts reading PCM from it.
func (r *Router) NewSource() (*Source, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, errors.New("pcmsink: router closed")
	}
	r.seq++
	path := filepath.Join(r.dir, "pcm-"+strconv.Itoa(r.seq)+".fifo")
	if err := ensureFIFO(path); err != nil {
		return nil, err
	}
	file, err := openFIFOReader(path)
	if err != nil {
		return nil, err
	}
	src := &Source{path: path, file: file, notify: make(chan struct{}, 1)}
	r.sources[src] = struct{}{}
	r.wg.Add(1)
	go r.readLoop(src)
	return src, nil
}

// RemoveSource detaches a source and removes its FIFO.
func (r *Router) RemoveSource(src *Source) {
	r.mu.Lock()
	if _, ok := r.sources[src]; !ok {
		r.mu.Unlock()
		return
	}
	delete(r.sources, src)
	r.mu.Unlock()

	src.mu.Lock()
	src.dead = true
	src.mu.Unlock()
	_ = src.file.Close()
	_ = os.Remove(src.path)
}

func (r *Router) readLoop(src *Source) {
	defer r.wg.Done()
	chunk := make([]byte, chunkBytes)
	for {
		n, err := src.file.Read(chunk)
		if n > 0 {
			src.push(chunk[:n])
			select {
			case r.dataCh <- src:
			default:
			}
		}
		if err != nil {
			select {
			case <-r.done:
				return
			default:
			}
			src.mu.Lock()
			dead := src.dead
			src.mu.Unlock()
			if dead {
				return
			}
			// EAGAIN, or EOF while no writer is attached. Wait and retry.
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if n == 0 {
			select {
			case <-r.done:
				return
			case <-time.After(10 * time.Millisecond):
			}
		}
	}
}

// mixLoop wakes when any source produced data, drains every source that has
// audio, mixes, and writes one chunk to the fanout.
func (r *Router) mixLoop() {
	defer r.wg.Done()
	for {
		select {
		case <-r.done:
			return
		case <-r.dataCh:
		}
		for {
			mixed := r.mixOnce()
			if mixed == nil {
				break
			}
			r.fanout.Write(mixed)
		}
	}
}

// mixOnce drains every source once and returns the mixed chunk, or nil when
// all sources are empty.
func (r *Router) mixOnce() []byte {
	r.mu.Lock()
	sources := make([]*Source, 0, len(r.sources))
	for s := range r.sources {
		sources = append(sources, s)
	}
	r.mu.Unlock()

	var chunks [][]byte
	for _, s := range sources {
		if b := s.drain(chunkBytes); len(b) > 0 {
			chunks = append(chunks, b)
		}
	}
	switch len(chunks) {
	case 0:
		return nil
	case 1:
		return chunks[0]
	}
	return mixS16(chunks)
}

// mixS16 sums s16le samples across streams with clamping. Frames past the
// shortest stream are passed through unmixed.
func mixS16(chunks [][]byte) []byte {
	longest := chunks[0]
	shortest := len(chunks[0])
	for _, c := range chunks[1:] {
		if len(c) > len(longest) {
			longest = c
		}
		shortest = min(shortest, len(c))
	}
	out := append([]byte(nil), longest...)
	for i := 0; i+1 < shortest; i += 2 {
		sum := int32(0)
		for _, c := range chunks {
			sum += int32(int16(binary.LittleEndian.Uint16(c[i:])))
		}
		sum = min(max(sum, math.MinInt16), math.MaxInt16)
		binary.LittleEndian.PutUint16(out[i:], uint16(int16(sum)))
	}
	return out
}

// Close removes all sources, stops the loops, and deletes the FIFO dir.
func (r *Router) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	sources := make([]*Source, 0, len(r.sources))
	for s := range r.sources {
		sources = append(sources, s)
	}
	r.mu.Unlock()

	close(r.done)
	for _, s := range sources {
		r.RemoveSource(s)
	}
	r.wg.Wait()
	_ = os.RemoveAll(r.dir)
}
