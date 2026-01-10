package group

import (
	"time"

	"github.com/WuCoNan/MyCache/store"
)

type group struct {
	Name       string
	localCache store.Store
}

type ByteView struct {
	data []byte
}

func (v ByteView) Len() int {
	return len(v.data)
}

func (v ByteView) cloneBytes() []byte {
	return append([]byte{}, v.data...)
}

func newGroupWithDefault(name string, cacheType store.CacheType) *group {
	g := &group{
		Name:       name,
		localCache: store.NewStore(cacheType, store.DefaultStoreOptions()),
	}
	return g
}

func (g *group) Get(key string) ([]byte, bool) {
	value, ok := g.localCache.Get(key)
	if !ok {
		return nil, false
	}
	return value.(ByteView).cloneBytes(), true
}

func (g *group) SetWithExpiration(key string, value []byte, duration time.Duration) error {
	return g.localCache.SetWithExpiration(key, ByteView{data: value}, duration)
}

func (g *group) Set(key string, value []byte) error {
	return g.localCache.Set(key, ByteView{data: value})
}

func (g *group) Delete(key string) {
	g.localCache.Delete(key)
}
