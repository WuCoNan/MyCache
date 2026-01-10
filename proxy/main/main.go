package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/WuCoNan/MyCache/proxy"
)

var (
	serviceName string
	ipAddr      string
	port        int
	connNum     int
)

func main() {
	flag.StringVar(&serviceName, "service_name", "mycache", "etcd service name")
	flag.StringVar(&ipAddr, "ip", "127.0.0.1", "ip address")
	flag.IntVar(&port, "port", 9090, "port number")
	flag.IntVar(&connNum, "conn_num", 5, "number of connections per cache node")
	flag.Parse()

	addr := ipAddr + ":" + fmt.Sprintf("%d", port)
	proxyServer := proxy.NewProxyServer(addr, serviceName, connNum)
	if err := proxyServer.Start(); err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}
}
