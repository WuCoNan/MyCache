package group

import (
	"sync"

	"github.com/WuCoNan/MyCache/store"
)

var (
	groupMgr  *groupManager
	groupOnce sync.Once
)

type groupManager struct {
	groups sync.Map // map[string]*group

}

func GetGroupManager() *groupManager {
	groupOnce.Do(func() {
		groupMgr = &groupManager{}
	})
	return groupMgr
}

func (gm *groupManager) GetGroup(name string) (*group, bool) {
	g, ok := gm.groups.Load(name)
	if !ok {
		return nil, false
	}
	return g.(*group), true
}

func (gm *groupManager) AddGroupWithDefault(name string) *group {
	g, _ := gm.groups.LoadOrStore(name, newGroupWithDefault(name, store.LRU))
	return g.(*group)
}

func (gm *groupManager) DeleteGroup(name string) {
	gm.groups.Delete(name)
}
