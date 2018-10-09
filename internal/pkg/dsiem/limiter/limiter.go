package limiter

import (
	"context"

	"golang.org/x/time/rate"
)

// Limiter provides rate per second control
type Limiter struct {
	lmt         *rate.Limiter
	maxRPS      int
	minRPS      int
	changeValue int
}

// New returns initialized Limiter
func New(maxRPS, minRPS int) *Limiter {
	l := new(Limiter)
	initial := maxRPS - ((maxRPS - minRPS) / 2)
	l.lmt = rate.NewLimiter(rate.Limit(initial), maxRPS)
	l.maxRPS = maxRPS
	l.minRPS = minRPS
	return l
}

func (l *Limiter) modifyLimit(raise bool) int {
	target := 0
	current := l.GetLimit()
	if raise {
		// raise to maxRPS in 100 steps
		target = current + ((l.maxRPS - l.minRPS) / 100)
		if target > l.maxRPS {
			target = l.maxRPS
		}
	} else {
		// lower to minRPS in 10 steps
		target = current - ((l.maxRPS - l.minRPS) / 10)
		if target < l.minRPS {
			target = l.minRPS
		}
	}
	l.lmt.SetLimit(rate.Limit(target))
	return target
}

// GetLimit returns the current RPS
func (l *Limiter) GetLimit() int {
	return int(l.lmt.Limit())
}

// RaiseLimit increase the RPS
func (l *Limiter) RaiseLimit() int {
	return l.modifyLimit(true)
}

// LowerLimit reduces the RPS
func (l *Limiter) LowerLimit() int {
	return l.modifyLimit(false)
}

// Wait returns the rate.limiter Wait function
func (l *Limiter) Wait(ctx context.Context) error {
	return l.lmt.Wait(ctx)
}
