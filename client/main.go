package main

import (
	"context"
	"encoding/json"
	// "fmt"
	"log"
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"
	"time"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// const (
// 	serverAddr = "localhost:23455"
// )

var (
	relayAddr = ""
	clientLogger *utils.Logger
)


type RelayNode struct {
	Address string `json:"address"`
	PubKey string `json:"pub_key"`
	Load int `json:"load"`
}

func getAvailableRelayNodes(etcdClient *clientv3.Client) ([]RelayNode, error) {
	var nodes []RelayNode
	resp, err := etcdClient.Get(context.Background(), utils.EtcdKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to fetch relay nodes: %v", err)
		return nodes, err
	}
	for _, ev := range resp.Kvs {
		var node RelayNode 
		err := json.Unmarshal(ev.Value, &node)
		if err != nil {
			log.Printf("Failed to decode relay node data: %v", err)
			continue
		}
		nodes = append(nodes, node)
	}
	log.Printf("%v", nodes)
	return nodes, nil
}


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

func main(){
	creds := utils.LoadCredentialsAsClient("certificates/ca.crt", 
											"certificates/client.crt",
											"certificates/client.key")

	etcdClient, err := initEtcdClient()
	if err != nil {
		log.Fatalf("Failed to initialize etcd client: %v", err)
	}
	defer etcdClient.Close()
	err = checkEtcdStatus(etcdClient)
	if err != nil {
		log.Fatalf("Etcd Server is unreachable: %v", err)
	}
	
	nodes, err := getAvailableRelayNodes(etcdClient)
	if err != nil {
		log.Fatalf("error while fetching avaliable relays: %v", err)
	}
	if len(nodes) > 0 {
		relayAddr = nodes[0].Address
	}
	
	clientLogger = utils.NewLogger("logs/client")
	conn, err := grpc.NewClient(relayAddr, grpc.WithTransportCredentials(creds))
	
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()

	client := routingpb.NewRelayNodeServerClient(conn)
	req := &routingpb.DummyRequest{Message: "Hi, This is Client"}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)
}