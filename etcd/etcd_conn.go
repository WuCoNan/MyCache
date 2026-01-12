package etcd

import (
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdOnce sync.Once
	etcdCli  *clientv3.Client
)

func GetEtcdClient() (*clientv3.Client, error) {
	etcdOnce.Do(func() {
		var err error
		etcdCli, err = clientv3.New(clientv3.Config{
			Endpoints:   []string{"localhost:2379"},
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			log.Printf("Failed to connect to etcd: %v", err)
			panic(err)
		}
	})
	return etcdCli, nil
}
