package store

import "time"

type Options struct {
	MaxBytes        int64
	CleanupInterval time.Duration
	OnEvict         func(key string, value Value)
}

type CacheType string

const (
	LRU  CacheType = "lru"
	LRU2 CacheType = "lru2"
)

type Store interface {
	Get(key string) (Value, bool)
	Set(key string, value Value) error
	SetWithExpiration(key string, value Value, duration time.Duration) error
	Delete(key string)
	Len() int
	Clear()
	Close()
}

func DefaultStoreOptions() Options {
	return Options{
		MaxBytes:        8 * 1024 * 1024, // 8MB
		CleanupInterval: time.Minute,
		OnEvict:         nil,
	}
}
func NewStore(cacheType CacheType, opts Options) Store {
	switch cacheType {
	case LRU:
		return newLRUCache(opts)
	default:
		return newLRUCache(opts)
	}
}
