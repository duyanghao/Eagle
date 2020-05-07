// Copyright 2020 duyanghao
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lrucache

import (
	"container/list"
	"errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback func(string)

type LruCache struct {
	sync.RWMutex
	limitSize   int64
	currentSize int64
	evictList   *list.List
	items       map[string]*list.Element
	onEvict     EvictCallback
}

// Entry is used to hold a value in the evictList
type entry struct {
	key   string
	value Entry
}

type Entry struct {
	Done      chan struct{}
	Completed bool
	Size      int64
}

// NewLRU constructs an LRU of the given size
func NewLRU(size int64, onEvict EvictCallback) (*LruCache, error) {
	if size <= 0 {
		return nil, errors.New("Must provide a positive size")
	}
	c := &LruCache{
		limitSize: size,
		evictList: list.New(),
		items:     make(map[string]*list.Element),
		onEvict:   onEvict,
	}
	return c, nil
}

// Get looks up a key's value from the cache
func (c *LruCache) Get(key string) (Entry, bool) {
	c.RLock()
	defer c.RUnlock()
	if ent, ok := c.items[key]; ok {
		if ent.Value.(*entry).value.Completed {
			c.evictList.MoveToFront(ent)
		}
		return ent.Value.(*entry).value, true
	}
	return Entry{}, false
}

// Create if not exists. Returns true if the entry existed.
func (c *LruCache) CreateIfNotExists(key string) (Entry, bool) {
	c.Lock()
	defer c.Unlock()
	// Check for existing item
	if ent, ok := c.items[key]; ok {
		if ent.Value.(*entry).value.Completed {
			c.evictList.MoveToFront(ent)
		}
		return ent.Value.(*entry).value, true
	}
	// Create new item
	ent := &entry{
		key: key,
		value: Entry{
			Done:      make(chan struct{}),
			Completed: false,
		},
	}
	c.items[key] = &list.Element{Value: ent}
	return ent.value, false
}

// removeElement is used to remove a given list element from the cache
func (c *LruCache) removeElement(e *list.Element) {
	c.currentSize -= e.Value.(*entry).value.Size
	c.evictList.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
	if c.onEvict != nil {
		c.onEvict(kv.key)
	}
}

// removeOldest removes the oldest item from the cache.
func (c *LruCache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// SetComplete mark completed status of cache entry.  Returns true if an eviction occurred.
// This function only works when entry exists
func (c *LruCache) SetComplete(key string, size int64) (evicted bool) {
	c.Lock()
	defer c.Unlock()
	// Check for existing item
	ent, ok := c.items[key]
	if !ok {
		return false
	}
	// Set status and size
	ent.Value.(*entry).value.Completed = true
	ent.Value.(*entry).value.Size = size
	close(ent.Value.(*entry).value.Done)

	entry := c.evictList.PushFront(ent.Value.(*entry))
	c.items[key] = entry

	// Verify size not exceeded
	c.currentSize += size
	evict := c.currentSize > c.limitSize
	if evict {
		c.removeOldest()
	}
	return evict
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LruCache) Remove(key string) (present bool) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		close(ent.Value.(*entry).value.Done)
		c.removeElement(ent)
		return true
	}
	return false
}

func (c *LruCache) Output() {
	c.RLock()
	defer c.RUnlock()
	for k, v := range c.items {
		log.Debugf("cache key: %s and value: %+v", k, v.Value.(*entry).value)
	}
	log.Debugf("cache size current/limit = %d/%d", c.currentSize, c.limitSize)
}
