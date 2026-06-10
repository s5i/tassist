//go:build windows

package exp

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/s5i/tassist/settings"
)

type CacheStats struct {
	Level        int
	TotalExp     int
	RemainingExp int

	SessionDelta    int
	SessionRate     int
	SessionDuration time.Duration

	Running bool
	Paused  bool
}

func NewCache(tmpDir string, stStorage *settings.Storage) (*Cache, error) {
	preset := stStorage.Preset()

	r, err := NewReader(tmpDir, preset.ClientWindowTitle)
	if err != nil {
		return nil, err
	}

	return &Cache{
		reader:       r,
		samplePeriod: time.Second,
	}, nil
}

func (c *Cache) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = true
	c.paused = false
	c.reset()
}

func (c *Cache) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = false
	c.paused = false
	c.reset()
}

func (c *Cache) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.paused = true
	if c.lastSample != nil && c.startSample != nil {
		c.stashedExp += c.lastSample.exp - c.startSample.exp
		c.stashedDur += c.lastSample.t.Sub(c.startSample.t)
	}
	c.lastSample = nil
	c.startSample = nil
}

func (c *Cache) Unpause() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.lastSample = nil
	c.startSample = nil
	c.paused = false
}

func (c *Cache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reset()
}

func (c *Cache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := CacheStats{
		Running: c.running,
		Paused:  c.paused,
	}

	dExp := c.stashedExp
	dDur := c.stashedDur

	if c.lastSample != nil {
		dExp += c.lastSample.exp - c.startSample.exp
		dDur += c.lastSample.t.Sub(c.startSample.t)

		s.TotalExp = c.lastSample.exp
		s.Level = c.level
		s.RemainingExp = c.nextLevelExp - c.lastSample.exp
	}

	dDur = dDur.Truncate(time.Second)
	if dDur >= time.Second {
		s.SessionRate = int(float64(dExp) * float64(time.Hour) / float64(dDur))
	}

	s.SessionDelta = dExp
	s.SessionDuration = dDur

	return s
}

func (c *Cache) Run(ctx context.Context) error {
	sampleTicker := time.NewTicker(c.samplePeriod)
	defer sampleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-sampleTicker.C:
			c.maybeGrabSample()
		}
	}
}

type Cache struct {
	reader *Reader
	mu     sync.Mutex

	startSample  *sample
	lastSample   *sample
	samplePeriod time.Duration

	stashedExp int
	stashedDur time.Duration

	level        int
	nextLevelExp int

	running bool
	paused  bool
}

func (c *Cache) maybeGrabSample() {
	c.mu.Lock()
	shouldSample := c.running && !c.paused
	c.mu.Unlock()

	if !shouldSample {
		return
	}

	exp, ok, err := c.reader.Read()
	if err != nil {
		log.Printf("exp.Reader.Read() failed: %v", err)
		return
	}
	if !ok {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// We might've stopped or paused during the read. Check again.
	shouldSample = c.running && !c.paused
	if !shouldSample {
		return
	}

	c.lastSample = &sample{exp, time.Now()}
	if c.startSample == nil {
		c.startSample = c.lastSample
	}

	if c.lastSample.exp < c.startSample.exp {
		c.reset()
		c.lastSample = &sample{exp, time.Now()}
		c.startSample = c.lastSample
	}

	if c.lastSample.exp >= c.nextLevelExp {
		for x := c.level + 1; ; x++ {
			expNeeded := (x*x*x - 6*x*x + 17*x - 12) * 50 / 3
			if expNeeded > c.lastSample.exp {
				c.level = x - 1
				c.nextLevelExp = expNeeded
				break
			}
		}
	}
}

func (c *Cache) reset() {
	c.lastSample = nil
	c.startSample = nil
	c.stashedExp = 0
	c.stashedDur = 0
	c.level = 0
	c.nextLevelExp = 0
}

type sample struct {
	exp int
	t   time.Time
}
