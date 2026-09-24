package loginguard

import (
	"sync"
	"time"
)

// Limiter is a sliding-window counter per key: at most max hits inside any
// window. It is in memory, so limits reset on restart and are per process -
// fine for one instance of this app, and the wrong tool if it is ever
// scaled out behind a balancer.
type Limiter struct {
	mu        sync.Mutex
	max       int
	window    time.Duration
	hits      map[string][]time.Time
	lastSweep time.Time
}

// maxKeys bounds memory: past it, every call sweeps expired keys instead of
// waiting for the periodic sweep.
const maxKeys = 50000

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, hits: map[string][]time.Time{}, lastSweep: time.Now()}
}

// prune drops key's hits that have left the window. Caller holds l.mu.
func (l *Limiter) prune(key string, now time.Time) []time.Time {
	h := l.hits[key]
	cutoff := now.Add(-l.window)
	i := 0
	for i < len(h) && !h[i].After(cutoff) {
		i++
	}
	if i > 0 {
		h = h[i:]
		if len(h) == 0 {
			delete(l.hits, key)
			return nil
		}
		l.hits[key] = h
	}
	return h
}

// Blocked reports whether key is at its limit and, if so, how long until
// its oldest hit leaves the window.
func (l *Limiter) Blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	h := l.prune(key, now)
	if len(h) < l.max {
		return false, 0
	}
	return true, h[0].Add(l.window).Sub(now)
}

// Record counts one hit against key.
func (l *Limiter) Record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.hits) > maxKeys || now.Sub(l.lastSweep) > l.window {
		for k := range l.hits {
			l.prune(k, now)
		}
		l.lastSweep = now
	}
	l.hits[key] = append(l.prune(key, now), now)
}

// Reset forgets key's hits.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
