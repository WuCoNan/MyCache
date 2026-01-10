package store

import (
	"fmt"
	"testing"
	"time"
)

type MyString string

func (str MyString) Len() int {
	return len(str)
}

var (
	lru     *lruCache
	options = Options{
		MaxBytes:        1000,
		CleanupInterval: time.Minute,
	}
)

func TestSet(t *testing.T) {

	if lru == nil {
		lru = newLRUCache(options)
	}
	lru.Set("Only", MyString("Apple can do"))
	fmt.Println("TestSet	:	Set successfully!")
}

func TestGet(t *testing.T) {
	if lru == nil {
		lru = newLRUCache(options)
	}
	if val, ok := lru.Get("Only"); ok {
		fmt.Printf("TestGet		:	Get  \"Only\":%s successfully \n", val)
	} else {
		fmt.Println("TestGet	:	Get  \"Only\"	failed")
	}

}
