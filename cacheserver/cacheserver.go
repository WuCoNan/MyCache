package cacheserver

import (
	"context"
	"log"
	"net"

	"github.com/WuCoNan/MyCache/etcd"
	"github.com/WuCoNan/MyCache/group"
	pb "github.com/WuCoNan/MyCache/pb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

type cacheServer struct {
	serviceName string
	addr        string
	nodeId      string
	stop        chan struct{}
	pb.UnimplementedCacheServiceServer
}

func NewCacheServer(serviceName, addr string, nodeId string) *cacheServer {
	return &cacheServer{
		serviceName: serviceName,
		addr:        addr,
		nodeId:      nodeId,
		stop:        make(chan struct{}),
	}
}

func (cs *cacheServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	gm := group.GetGroupManager()
	groupName, key := req.GetGroupName(), req.GetKey()
	g, ok := gm.GetGroup(groupName)
	// If the group doesn't exist
	if !ok {
		return &pb.GetResponse{Value: nil, Flag: false}, nil
	}
	value, ok := g.Get(key)
	return &pb.GetResponse{Value: value, Flag: ok}, nil
}

func (cs *cacheServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	gm := group.GetGroupManager()
	groupName, key, value := req.GetGroupName(), req.GetKey(), req.GetValue()
	g, ok := gm.GetGroup(groupName)
	// If the group doesn't exist
	if !ok {
		return &pb.SetResponse{Flag: false}, nil
	}
	err := g.Set(key, value)
	// If setting the value fails
	if err != nil {
		return &pb.SetResponse{Flag: false}, nil
	}
	return &pb.SetResponse{Flag: true}, nil
}

func (cs *cacheServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	gm := group.GetGroupManager()
	groupName, key := req.GetGroupName(), req.GetKey()
	g, ok := gm.GetGroup(groupName)
	// If the group doesn't exist
	if !ok {
		return &pb.DeleteResponse{Flag: false}, nil
	}
	g.Delete(key)
	return &pb.DeleteResponse{Flag: true}, nil
}

func (cs *cacheServer) AddGroup(ctx context.Context, req *pb.AddGroupRequest) (*pb.AddGroupResponse, error) {
	gm := group.GetGroupManager()
	groupName := req.GetGroupName()
	gm.AddGroupWithDefault(groupName)
	return &pb.AddGroupResponse{Flag: true}, nil

}
func (cs *cacheServer) Start() error {
	listen, err := net.Listen("tcp", cs.addr)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	pb.RegisterCacheServiceServer(s, cs)

	go cs.Register()

	log.Printf("Cache server %s started at %s", cs.nodeId, cs.addr)
	if err := s.Serve(listen); err != nil {
		return err
	}
	return nil
}

func (cs *cacheServer) Stop() {
	close(cs.stop)
}

func (cs *cacheServer) Register() error {
	log.Printf("Registering cache server %s with etcd", cs.nodeId)
	etcdClient, err := etcd.GetEtcdClient()

	if err != nil {
		log.Printf("Failed to get etcd client: %v", err)
		return err
	}
	resp, err := etcdClient.Grant(context.TODO(), 5)
	if err != nil {
		log.Printf("Failed to grant lease: %v", err)
		return err
	}
	_, err = etcdClient.Put(context.TODO(), "/services/"+cs.serviceName+"/"+cs.nodeId, cs.addr, clientv3.WithLease(resp.ID))

	if err != nil {
		log.Printf("Failed to register service with etcd: %v", err)
		return err
	}

	ch, err := etcdClient.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Printf("Failed to set up keep-alive: %v", err)
		return err
	}
	go func() {
		for {
			select {
			case <-ch:
				//log.Printf("Cache server %s sent keep-alive", cs.nodeId)
			case <-cs.stop:
				return
			}
		}
	}()
	return nil
}
