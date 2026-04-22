//go:build windows

package exp

import (
	"context"
	"sync"
	"time"
)

func NewCache() (*Cache, error) {
	r, err := NewReader()
	if err != nil {
		return nil, err
	}

	return &Cache{
		reader:       r,
		samples:      map[time.Time]int{},
		samplePeriod: 5 * time.Second,
		prunePeriod:  time.Hour,
		pruneAge:     6 * time.Hour,
	}, nil
}

func (c *Cache) Latest() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	latest, ok := c.samples[c.latest]
	return latest, ok
}

func (c *Cache) Delta(window time.Duration) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	latest := c.samples[c.latest]
	prev, ok := c.samples[c.latest.Add(-window).Truncate(c.samplePeriod)]

	if !ok {
		return 0, false
	}

	return latest - prev, true
}

func (c *Cache) Reset() {
	c.mu.Lock()
	c.samples = map[time.Time]int{}
	c.mu.Unlock()

	c.addSample()
}

func (c *Cache) Run(ctx context.Context) error {
	pruneTicker := time.NewTicker(c.prunePeriod)
	defer pruneTicker.Stop()

	sampleTicker := time.NewTicker(c.samplePeriod)
	defer sampleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pruneTicker.C:
			c.prune()
		case <-sampleTicker.C:
			c.addSample()
		}
	}
}

type Cache struct {
	reader       *Reader
	samples      map[time.Time]int
	latest       time.Time
	mu           sync.Mutex
	pruneAge     time.Duration
	samplePeriod time.Duration
	prunePeriod  time.Duration
}

func (c *Cache) addSample() {
	exp, ok, err := c.reader.Read()
	if err != nil || !ok {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Truncate(c.samplePeriod)
	c.samples[now] = exp
	c.latest = now
}

func (c *Cache) prune() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Truncate(c.samplePeriod)
	for t := range c.samples {
		if t.Add(c.pruneAge).Before(now) {
			delete(c.samples, t)
		}
	}
}
