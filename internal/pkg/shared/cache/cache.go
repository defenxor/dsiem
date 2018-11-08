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

package cache

import (
	"time"

	"github.com/allegro/bigcache"
)

// Cache wraps bigcache
type Cache struct {
	ID    string
	cache *bigcache.BigCache
}

// New returns initialized Cache
func New(name string, lifetimeMinutes int, shards int) (*Cache, error) {
	c := Cache{}
	c.ID = name
	// default to 10 minutes
	if lifetimeMinutes == 0 {
		lifetimeMinutes = 10
	}
	if shards == 0 {
		shards = 128
	}
	config := bigcache.Config{
		Shards:     shards,                                       // number of shards (must be a power of 2)
		LifeWindow: time.Duration(lifetimeMinutes) * time.Minute, // time after which entry can be evicted
		// rps * lifeWindow, used only in initial
		// memory allocation
		MaxEntriesInWindow: shards * lifetimeMinutes * 60,
		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: 500,
		// prints information about additional memory allocation
		Verbose: false,
		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: shards,
		// callback fired when the oldest entry is removed because of its expiration time or no space left
		// for the new entry, or because delete was called. A bitmask representing the reason will be returned.
		// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
		OnRemove: nil,
		// OnRemoveWithReason is a callback fired when the oldest entry is removed because of its expiration time or no space left
		// for the new entry, or because delete was called. A constant representing the reason will be passed through.
		// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
		// Ignored if OnRemove is specified.
		OnRemoveWithReason: nil,
	}

	p, err := bigcache.NewBigCache(config)
	if err != nil {
		return nil, err
	}
	c.cache = p
	return &c, nil
}

// Set store the key value in cache
func (c *Cache) Set(key string, value []byte) {
	c.cache.Set(key, value)
}

// Get returns value of key from cache
func (c *Cache) Get(key string) (value []byte, err error) {
	value, err = c.cache.Get(key)
	return
}
