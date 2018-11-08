// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package limiter

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter provides rate per second control
type Limiter struct {
	sync.RWMutex
	lmt         *rate.Limiter
	maxRPS      int
	minRPS      int
	changeValue int
}

// New returns initialized Limiter
func New(maxRPS, minRPS int) (*Limiter, error) {
	if minRPS > maxRPS {
		return nil, errors.New("minRPS must be <= maxRPS")
	}
	l := new(Limiter)
	l.Lock()
	defer l.Unlock()

	initial := maxRPS - ((maxRPS - minRPS) / 2)
	l.lmt = rate.NewLimiter(rate.Limit(initial), maxRPS)
	l.maxRPS = maxRPS
	l.minRPS = minRPS
	return l, nil
}

func (l *Limiter) modifyLimit(raise bool) int {
	target := 0
	current := l.Limit()
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

// Limit returns the current RPS
func (l *Limiter) Limit() int {
	l.RLock()
	defer l.RUnlock()
	return int(l.lmt.Limit())
}

// Raise increase the RPS
func (l *Limiter) Raise() int {
	return l.modifyLimit(true)
}

// Lower reduces the RPS
func (l *Limiter) Lower() int {
	return l.modifyLimit(false)
}

// Wait returns the rate.limiter Wait function
func (l *Limiter) Wait(ctx context.Context) error {
	return l.lmt.Wait(ctx)
}
