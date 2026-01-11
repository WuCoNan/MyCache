package client

import (
	"context"
	"log"

	pb "github.com/WuCoNan/MyCache/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn      *grpc.ClientConn
	client    pb.ProxyServiceClient
	proxyAddr string
}

func NewClient(addr string) *Client {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to proxy server: %v", err)
		return nil
	}
	return &Client{
		conn:      conn,
		client:    pb.NewProxyServiceClient(conn),
		proxyAddr: addr,
	}
}

func (c *Client) Get(groupName string, key string) ([]byte, bool) {
	resp, err := c.client.Get(context.TODO(), &pb.GetRequest{
		GroupName: groupName,
		Key:       key,
	})
	if err != nil {
		log.Printf("Get request failed: %v", err)
		return nil, false
	}
	return resp.Value, resp.Flag
}

func (c *Client) Set(groupName string, key string, value []byte) bool {
	resp, err := c.client.Set(context.TODO(), &pb.SetRequest{
		GroupName: groupName,
		Key:       key,
		Value:     value,
	})
	if err != nil {
		log.Printf("Set request failed: %v", err)
		return false
	}
	return resp.Flag
}

func (c *Client) Delete(groupName string, key string) bool {
	resp, err := c.client.Delete(context.TODO(), &pb.DeleteRequest{
		GroupName: groupName,
		Key:       key,
	})
	if err != nil {
		log.Printf("Delete request failed: %v", err)
		return false
	}
	return resp.Flag
}

func (c *Client) AddGroup(groupName string) bool {
	resp, err := c.client.AddGroup(context.TODO(), &pb.AddGroupRequest{
		GroupName: groupName,
	})
	if err != nil {
		log.Printf("AddGroup request failed: %v", err)
		return false
	}
	return resp.Flag
}
