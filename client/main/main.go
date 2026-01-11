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
	key := "exampleKey"
	value := []byte("exampleValue")
	if ok := c.Set(groupName, key, value); !ok {
		return
	}
	val, ok := c.Get(groupName, key)
	log.Printf("Get key: %s, value: %s, success: %v", key, string(val), ok)

	if ok := c.Delete(groupName, key); !ok {
		log.Printf("Failed to delete key: %s", key)
		return
	}
}
