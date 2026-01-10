package main

import (
	"flag"
	"fmt"

	"github.com/WuCoNan/MyCache/cacheserver"
)

var (
	serviceName string
	ipAddr      string
	port        int
	nodeId      string
)

func main() {
	flag.StringVar(&serviceName, "service_name", "mycache", "etcd service name")
	flag.StringVar(&ipAddr, "ip", "127.0.0.1", "ip address")
	flag.IntVar(&port, "port", 8080, "port number")
	flag.StringVar(&nodeId, "node", "A", "node id")
	flag.Parse()
	addr := ipAddr + ":" + fmt.Sprintf("%d", port)
	cacheServer := cacheserver.NewCacheServer(serviceName, addr, nodeId)

	if err := cacheServer.Start(); err != nil {
		panic(err)
	}
}
