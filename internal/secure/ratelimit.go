package secure

import (
	"sync"
	"time"
)

// Limiter counts failed attempts per key (user name, IP) in a sliding window and
// blocks the key for a while once the limit is exceeded. In-memory only.
type Limiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	lockout time.Duration
	m       map[string]*entry
	now     func() time.Time
}

type entry struct {
	fails  []time.Time
	locked time.Time
}

func NewLimiter(max int, window, lockout time.Duration) *Limiter {
	return &Limiter{max: max, window: window, lockout: lockout, m: map[string]*entry{}, now: time.Now}
}

// Blocked reports whether key is locked and for how much longer.
func (l *Limiter) Blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.m[key]
	if e == nil {
		return false, 0
	}
	if d := e.locked.Sub(l.now()); d > 0 {
		return true, d
	}
	return false, 0
}

// Fail records a failure; returns true if the key just became locked.
func (l *Limiter) Fail(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e := l.m[key]
	if e == nil {
		e = &entry{}
		l.m[key] = e
	}
	cut := now.Add(-l.window)
	kept := e.fails[:0]
	for _, t := range e.fails {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	e.fails = append(kept, now)
	if len(e.fails) >= l.max {
		e.locked = now.Add(l.lockout)
		e.fails = e.fails[:0]
		return true
	}
	return false
}

// Reset clears the key after a successful login.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.m, key)
	l.mu.Unlock()
}

// Cleanup drops stale entries; call periodically.
func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for k, e := range l.m {
		if e.locked.Before(now) && (len(e.fails) == 0 || e.fails[len(e.fails)-1].Before(now.Add(-l.window))) {
			delete(l.m, k)
		}
	}
}
