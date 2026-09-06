// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package jukebox

import (
	"sync"
	"time"
)

type Track struct {
	ID       string
	Title    string
	Artist   string
	Album    string
	Duration int
	CoverArt string
}

type Status struct {
	Current     *Track  `json:"current,omitempty"`
	Playing     bool    `json:"playing"`
	Gain        float64 `json:"gain"`
	PositionSec int     `json:"positionSec"`
	Queue       []Track `json:"queue,omitempty"`
	UpdatedAt   string  `json:"updatedAt"`
}

type Controller struct {
	mu          sync.RWMutex
	current     *Track
	playing     bool
	gain        float64
	positionSec int
	queue       []Track
	updatedAt   time.Time
}

func NewController() *Controller {
	return &Controller{gain: 1.0, updatedAt: time.Now()}
}

func (c *Controller) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := Status{
		Playing:     c.playing,
		Gain:        c.gain,
		PositionSec: c.positionSec,
		UpdatedAt:   c.updatedAt.UTC().Format(time.RFC3339),
	}
	if c.current != nil {
		copyTrack := *c.current
		out.Current = &copyTrack
	}
	if len(c.queue) > 0 {
		out.Queue = append([]Track(nil), c.queue...)
	}
	return out
}

func (c *Controller) Control(action string, track *Track, positionSec int, gain float64, queue []Track) Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch action {
	case "start":
		c.playing = true
	case "stop":
		c.playing = false
		c.positionSec = 0
	case "skip":
		if len(c.queue) > 0 {
			next := c.queue[0]
			c.queue = c.queue[1:]
			c.current = &next
			c.positionSec = 0
			c.playing = true
		} else {
			c.playing = false
			c.current = nil
			c.positionSec = 0
		}
	case "set":
		if track != nil {
			copyTrack := *track
			c.current = &copyTrack
		}
		if positionSec >= 0 {
			c.positionSec = positionSec
		}
		if len(queue) > 0 {
			c.queue = append([]Track(nil), queue...)
		}
	case "setGain":
		if gain >= 0 {
			c.gain = gain
		}
	case "status":
		// no-op
	}
	c.updatedAt = time.Now()
	return c.snapshotLocked()
}

func (c *Controller) snapshotLocked() Status {
	out := Status{
		Playing:     c.playing,
		Gain:        c.gain,
		PositionSec: c.positionSec,
		UpdatedAt:   c.updatedAt.UTC().Format(time.RFC3339),
	}
	if c.current != nil {
		copyTrack := *c.current
		out.Current = &copyTrack
	}
	if len(c.queue) > 0 {
		out.Queue = append([]Track(nil), c.queue...)
	}
	return out
}
