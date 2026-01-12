package main

import (
	"log"

	"github.com/WuCoNan/MyCache/client"
)

func main() {
	c := client.NewClient("127.0.0.1:9090")
	if c == nil {
		return
	}

	groupName := "testGroup"
	if ok := c.AddGroup(groupName); !ok {
		return
	}
	keys := []string{"key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8", "key9", "key10"}
	for _, key := range keys {
		if ok := c.Set(groupName, key, []byte("exampleValue")); !ok {
			return
		}
		val, ok := c.Get(groupName, key)
		log.Printf("Get key: %s, value: %s, success: %v", key, string(val), ok)
		if ok := c.Delete(groupName, key); !ok {
			return
		}
		log.Printf("Deleted key: %s", key)
	}

}
