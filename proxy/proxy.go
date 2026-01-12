package proxy

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/WuCoNan/MyCache/etcd"
	pb "github.com/WuCoNan/MyCache/pb"
	"github.com/golang/groupcache/consistenthash"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type clientConn struct {
	conn   *grpc.ClientConn
	client pb.CacheServiceClient
}

type proxyServer struct {
	addr            string
	cacheSeriveName string
	connPool        sync.Map //map[string]chan *clientConn
	connNum         int
	consHash        *consistenthash.Map
	cacheNodeNum    int
	cacheNodeAddrs  map[string]struct{}
	pb.UnimplementedProxyServiceServer
}

func NewProxyServer(addr string, cacheSeriveName string, connNum int) *proxyServer {
	ps := &proxyServer{
		addr:            addr,
		cacheSeriveName: cacheSeriveName,
		connNum:         connNum,
		consHash:        consistenthash.New(50, nil),
		cacheNodeNum:    0,
		cacheNodeAddrs:  make(map[string]struct{}),
	}

	return ps
}

func (ps *proxyServer) Start() error {
	lis, err := net.Listen("tcp", ps.addr)
	if err != nil {
		log.Printf("Failed to listen on %s: %v", ps.addr, err)
		return err
	}
	ps.startDiscovery()

	grpcServer := grpc.NewServer()
	pb.RegisterProxyServiceServer(grpcServer, ps)
	log.Printf("Proxy server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		return err
	}
	return nil
}

func (ps *proxyServer) startDiscovery() {
	ps.fetchAllServices()
	go ps.watchService()
}

func (ps *proxyServer) fetchAllServices() {
	etcdClient, err := etcd.GetEtcdClient()
	if err != nil {
		log.Fatalf("Failed to get etcd client: %v", err)
		return
	}
	resp, err := etcdClient.Get(context.Background(), "/services/"+ps.cacheSeriveName+"/", clientv3.WithPrefix())
	if err != nil {
		log.Fatalf("Failed to fetch services from etcd: %v", err)
		return
	}
	for _, kv := range resp.Kvs {
		log.Printf("Discovered service: %s at %s", string(kv.Key), string(kv.Value))
		ps.createConns(string(kv.Value))
	}
}

func (ps *proxyServer) watchService() {
	etcdClient, err := etcd.GetEtcdClient()
	if err != nil {
		log.Fatalf("Failed to get etcd client: %v", err)
		return
	}

	rch := etcdClient.Watch(context.Background(), "/services/"+ps.cacheSeriveName+"/", clientv3.WithPrefix(), clientv3.WithPrevKV())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				log.Printf("New service added: %s at %s", string(ev.Kv.Key), string(ev.Kv.Value))
				ps.createConns(string(ev.Kv.Value))
			case clientv3.EventTypeDelete:
				log.Printf("Service removed: %s value: %s", string(ev.Kv.Key), string(ev.PrevKv.Value))
				ps.releaseConns(string(ev.PrevKv.Value))
				// Handle service removal if necessary
			}
		}
	}
}

func (ps *proxyServer) createConns(addr string) {
	if actual, loaded := ps.connPool.LoadOrStore(addr, make(chan *clientConn, ps.connNum)); loaded {
		log.Printf("Connections to %s already exist", addr)
	} else {
		ps.consHash.Add(string(addr))
		ps.cacheNodeNum++
		ps.cacheNodeAddrs[addr] = struct{}{}
		for i := 0; i < ps.connNum; i++ {
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to create gRPC connection to %s: %v", addr, err)
				continue
			}
			actual.(chan *clientConn) <- &clientConn{
				conn:   conn,
				client: pb.NewCacheServiceClient(conn),
			}
		}
	}
}

func (ps *proxyServer) releaseConns(addr string) {
	if ch, ok := ps.connPool.Load(addr); ok {
		close(ch.(chan *clientConn))
		ps.connPool.Delete(addr)
		delete(ps.cacheNodeAddrs, addr)
		ps.cacheNodeNum--

		ps.consHash = consistenthash.New(50, nil)
		for addr := range ps.cacheNodeAddrs {
			ps.consHash.Add(string(addr))
		}
	}
	log.Printf("Released connections to %s", addr)
}
func (ps *proxyServer) cachePicker(key string) string {
	log.Printf("Picking cache node for key: %s to %s", key, ps.consHash.Get(key))
	return ps.consHash.Get(key)
}

func (ps *proxyServer) getClientConn(addr string) *clientConn {
	if ch, ok := ps.connPool.Load(addr); ok {
		return <-ch.(chan *clientConn)
	}
	log.Printf("No connection found for address: %s", addr)
	return nil
}
func (ps *proxyServer) releaseClientConn(addr string, cc *clientConn) {
	if ch, ok := ps.connPool.Load(addr); ok {
		ch.(chan *clientConn) <- cc
	} else {
		log.Printf("No connection pool found for address: %s", addr)
	}
}

func (ps *proxyServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	key := req.GetKey()
	addr := ps.cachePicker(key)
	cc := ps.getClientConn(addr)
	if cc == nil {
		log.Printf("No client connection available for key: %s", key)
		return &pb.GetResponse{Value: nil, Flag: false}, nil
	}
	defer ps.releaseClientConn(addr, cc)

	return cc.client.Get(ctx, req)
}

func (ps *proxyServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	key := req.GetKey()
	addr := ps.cachePicker(key)
	cc := ps.getClientConn(addr)
	if cc == nil {
		log.Printf("No client connection available for key: %s", key)
		return &pb.SetResponse{Flag: false}, nil
	}
	defer ps.releaseClientConn(addr, cc)

	return cc.client.Set(ctx, req)
}

func (ps *proxyServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	key := req.GetKey()
	addr := ps.cachePicker(key)
	cc := ps.getClientConn(addr)
	if cc == nil {
		log.Printf("No client connection available for key: %s", key)
		return &pb.DeleteResponse{Flag: false}, nil
	}
	defer ps.releaseClientConn(addr, cc)
	return cc.client.Delete(ctx, req)
}

func (ps *proxyServer) AddGroup(ctx context.Context, req *pb.AddGroupRequest) (*pb.AddGroupResponse, error) {
	flagChList := make([]chan bool, ps.cacheNodeNum)
	addrs := make([]string, 0, ps.cacheNodeNum)
	for addr := range ps.cacheNodeAddrs {
		addrs = append(addrs, addr)
	}
	for i := 0; i < ps.cacheNodeNum; i++ {
		flagChList[i] = make(chan bool)

		go func(i int) {
			addr := addrs[i]
			cc := ps.getClientConn(addr)
			if cc == nil {
				log.Printf("No client connection available for address: %s", addr)
				flagChList[i] <- false
				return
			}
			defer ps.releaseClientConn(addr, cc)
			resp, err := cc.client.AddGroup(ctx, req)
			if err != nil {
				log.Printf("Error calling AddGroup on %s: %v", addr, err)
				flagChList[i] <- false
				return
			}
			flagChList[i] <- resp.GetFlag()
		}(i)
	}

	finalFlag := true
	for i := 0; i < ps.cacheNodeNum; i++ {
		flag := <-flagChList[i]
		finalFlag = finalFlag && flag
	}

	return &pb.AddGroupResponse{Flag: finalFlag}, nil
}
