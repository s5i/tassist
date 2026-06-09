package ping

import (
	"context"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/s5i/tassist/settings"
	"golang.org/x/sync/errgroup"
)

func New(stStorage *settings.Storage) (*Pinger, error) {
	ret := &Pinger{}

	st := stStorage.Preset()
	p, err := new(st.ServerAddr)
	if err != nil {
		return nil, err
	}
	ret.pinger = p

	for _, addr := range st.ProxyAddrs {
		p, err := new(addr)
		if err != nil {
			return nil, err
		}
		ret.proxyPingers = append(ret.proxyPingers, p)
	}

	return ret, nil
}

func new(addr string) (*pinger, error) {
	ret := &pinger{
		wait: make([]bool, 20),
	}

	pinger := probing.New(addr)
	pinger.SetPrivileged(true)
	pinger.OnRecv = func(p *probing.Packet) {
		ret.registerRecv(p.Rtt)
	}
	pinger.OnSend = func(p *probing.Packet) {
		ret.registerSend()
	}
	ret.pinger = pinger

	return ret, nil
}

type Pinger struct {
	pinger       *pinger
	proxyPingers []*pinger
}

func (m *Pinger) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return m.pinger.Run(ctx)
	})
	for _, p := range m.proxyPingers {
		eg.Go(func() error {
			return p.Run(ctx)
		})
	}
	return eg.Wait()
}

func (m *Pinger) Stats() Stats {
	ret := m.pinger.Stats()

	plNum := ret.PacketLoss
	plDen := 1
	for _, pp := range m.proxyPingers {
		if p := pp.Stats(); p.OK {
			plNum += p.PacketLoss
			plDen++
		}
	}

	ret.PacketLoss = plNum / float64(plDen)
	return ret
}

type pinger struct {
	pinger *probing.Pinger
	ok     bool
	rtt    time.Duration
	nFails int
	wait   []bool
	waitIt int
	mu     sync.Mutex
}

type Stats struct {
	OK         bool
	RTT        time.Duration
	PacketLoss float64
}

func (p *pinger) Run(ctx context.Context) error {
	return p.pinger.RunWithContext(ctx)
}

func (p *pinger) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()

	ret := Stats{
		OK:         p.ok,
		RTT:        p.rtt,
		PacketLoss: float64(p.nFails) / float64(len(p.wait)-1),
	}

	return ret
}

func (p *pinger) registerRecv(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ok = true
	p.rtt = d
	p.wait[p.waitIt] = false
}

func (p *pinger) registerSend() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Last recv didn't arrive.
	if p.wait[p.waitIt] {
		p.ok = false
		p.rtt = 0
		p.nFails++
	}

	p.waitIt = (p.waitIt + 1) % len(p.wait)

	// Discount the failure from the previous loop if present.
	if p.wait[p.waitIt] {
		p.nFails--
	}

	p.wait[p.waitIt] = true
}
