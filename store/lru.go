package store

import (
	"container/list"
	"sync"
	"time"
)

type Value interface {
	Len() int
}

type lruCache struct {
	mu              sync.RWMutex
	list            *list.List
	items           map[string]*list.Element
	expires         map[string]time.Time
	maxBytes        int64
	useBytes        int64
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker
	onEvict         func(key string, value Value)
	closeChan       chan struct{}
}

type lruEntry struct {
	key string
	val Value
}

func newLRUCache(opts Options) *lruCache {
	cleanupInterval := opts.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute
	}

	c := &lruCache{
		list:            list.New(),
		items:           make(map[string]*list.Element),
		expires:         make(map[string]time.Time),
		maxBytes:        opts.MaxBytes,
		cleanupInterval: cleanupInterval,
		cleanupTicker:   time.NewTicker(cleanupInterval),
		onEvict:         opts.OnEvict,
		closeChan:       make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}
func (c *lruCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list.Len()
}

func (c *lruCache) SetWithExpiration(key string, value Value, duration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if duration > 0 {
		c.expires[key] = time.Now().Add(duration)
	} else {
		delete(c.expires, key)
	}

	if elem, ok := c.items[key]; ok {
		c.list.MoveToBack(elem)
		entry := elem.Value.(*lruEntry)
		c.useBytes += int64(value.Len() - entry.val.Len())
		entry.val = value
		return nil
	}

	entry := &lruEntry{key, value}
	elem := c.list.PushBack(entry)
	c.items[key] = elem
	c.useBytes += int64(value.Len() + len(key))

	c.evict()
	return nil
}

func (c *lruCache) Set(key string, value Value) error {
	return c.SetWithExpiration(key, value, 0)
}

func (c *lruCache) Get(key string) (Value, bool) {
	c.mu.RLock()
	elem, ok := c.items[key]

	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	entry := elem.Value.(*lruEntry)

	if expiretime, exits := c.expires[key]; exits {
		if time.Now().After(expiretime) {
			c.mu.RUnlock()
			c.Delete(key)
			return nil, false
		}
	}

	c.mu.RUnlock()

	c.mu.Lock()
	if _, ok = c.items[key]; ok {
		c.list.MoveToBack(elem)
	}
	c.mu.Unlock()
	return entry.val, true

}

func (c *lruCache) evict() {
	now := time.Now()
	for key, expiretime := range c.expires {
		if now.After(expiretime) {
			c.removeElement(c.items[key])
		}
	}

	for c.useBytes > c.maxBytes && c.list.Len() > 0 {
		c.removeElement(c.list.Front())
	}
}

func (c *lruCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}
func (c *lruCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	delete(c.items, entry.key)
	delete(c.expires, entry.key)
	c.list.Remove(elem)
	c.useBytes -= int64(entry.val.Len() + len(entry.key))

	if c.onEvict != nil {
		c.onEvict(entry.key, entry.val)
	}
}

func (c *lruCache) cleanupLoop() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.mu.Lock()
			c.evict()
			c.mu.Unlock()
		case <-c.closeChan:
			return
		}
	}
}

func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, elem := range c.items {
		entry := elem.Value.(*lruEntry)
		value := entry.val
		if c.onEvict != nil {
			c.onEvict(key, value)
		}
	}

	c.list.Init()
	c.items = make(map[string]*list.Element)
	c.expires = make(map[string]time.Time)
	c.useBytes = 0
}

func (c *lruCache) UsedBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.useBytes
}

func (c *lruCache) MaxBytes() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxBytes
}

func (c *lruCache) SetMaxBytes(maxBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxBytes = maxBytes
	c.evict()
}

func (c *lruCache) Close() {
	close(c.closeChan)
	c.cleanupTicker.Stop()
}

func (c *lruCache) GetWithExpiration(key string) (Value, time.Duration, bool) {
	c.mu.RLock()
	if _, ok := c.items[key]; !ok {
		c.mu.RUnlock()
		return nil, 0, false
	}

	var ttl time.Duration
	if expire, ok := c.expires[key]; ok {
		if time.Now().After(expire) {
			c.mu.RUnlock()
			c.Delete(key)
			return nil, 0, false
		}
		ttl = time.Now().Sub(expire)
	}

	c.mu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, 0, false
	}
	c.list.MoveToBack(elem)

	if _, ok := c.expires[key]; ok {
		return elem.Value.(*lruEntry).val, ttl, true
	} else {
		return elem.Value.(*lruEntry).val, 0, true
	}
}

func (c *lruCache) GetExpiration(key string) (time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	expire, ok := c.expires[key]
	return expire, ok
}

func (c *lruCache) UpdateExpiration(key string, expiration time.Duration) bool {
	c.mu.RLock()
	if _, ok := c.expires[key]; !ok {
		return false
	}

	c.mu.RUnlock()
	c.mu.Lock()

	if _, ok := c.expires[key]; !ok {
		return false
	}
	c.expires[key] = c.expires[key].Add(expiration)
	return true
}
