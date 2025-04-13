package main

import (
	// "context"
	// "fmt"
	"context"
	"log"
	"time"
	"encoding/json"
	// "google.golang.org/protobuf/proto"

	utils "onion_routing/utils"

	"go.etcd.io/etcd/client/v3"

	// "google.golang.org/grpc/metadata"
)

func initEtcdClient()(*clientv3.Client, error){
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{utils.EtcdServerAddr},
		DialTimeout: utils.EtcdTimeOutInterval * time.Second,
	})
	return etcdClient, err
}

func checkEtcdStatus(etcdClient *clientv3.Client)(error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := etcdClient.Status(ctx, utils.EtcdServerAddr)
	return err
}

func createLease(etcdClient *clientv3.Client)(clientv3.LeaseID, error){
	leaseResp, err := etcdClient.Grant(context.Background(), utils.EtcdLeaseTTL)
	return leaseResp.ID, err
}

func registerWithEtcdServer(client *clientv3.Client, leaseID clientv3.LeaseID)(error){
	key := utils.EtcdKeyPrefix + nodeID
	relayNode := RelayNode{
		Address: relayAddr,
		PubKey: pubKey,
		Load: load,
	}
	data, _ := json.Marshal(relayNode)
	_, err := client.Put(context.Background(), key, string(data), clientv3.WithLease(leaseID))
	return err
}

func keepAliveThread(client *clientv3.Client, leaseID clientv3.LeaseID) {
	ch, err := client.KeepAlive(context.Background(), leaseID)
	if err != nil {
		log.Fatalf("Failed to keep alive: %v", err)
	}
	for range ch {}  // to consumed keepalive responses
}