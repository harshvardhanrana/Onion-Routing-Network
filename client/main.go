package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"crypto/rsa"

	routingpb "onion_routing/protofiles"
	encryption "onion_routing/encryption"
	utils "onion_routing/utils"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// const (
// 	serverAddr = "localhost:23455"
// )

var (
	clientLogger *utils.Logger
	nodes []RelayNode
)


type RelayNode struct {
	Address string `json:"address"`
	PubKey *rsa.PublicKey `json:"pub_key"`
	Load int `json:"load"`
}

func getAvailableRelayNodes(etcdClient *clientv3.Client) ([]RelayNode, error) {
	nodes = []RelayNode{}
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
	log.Printf("Nodes: %v", nodes)
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

func startCreationRoute(client routingpb.RelayNodeServerClient){
	cell := encryption.SampleCell()

	var message = encryption.BuildMessage(cell)

	fmt.Printf("Built Message:\n%x\n", message)

	fmt.Printf("Size of message: %d bytes\n", len(message))

	// pprint pubkey 
	fmt.Print("Public Key: ")
	fmt.Printf("%v\n", nodes[0].PubKey)

	encrypted_message, err := encryption.EncryptRSA(message, nodes[0].PubKey)

	req := &routingpb.DummyRequest{Message: encrypted_message}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)

}

//TODO: Immplement this function later
func GetNodesInRoute(nodes []RelayNode) (){

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
	
	GetNodesInRoute(nodes)
	
	clientLogger = utils.NewLogger("logs/client")
	conn, err := grpc.NewClient(nodes[0].Address, grpc.WithTransportCredentials(creds))
	
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()

	client := routingpb.NewRelayNodeServerClient(conn)

	startCreationRoute(client)
	// req := &routingpb.DummyRequest{Message: "Hi, This is Client"}

	// clientLogger.PrintLog("Request sending to server: %v", req)
	// resp, err := client.RelayNodeRPC(context.Background(), req)
	// if err != nil {
	// 	log.Fatalf("error whiling calling rpc: %v\n", err)
	// }

	// clientLogger.PrintLog("Response received from server: %v", resp)
	// log.Printf("Response received from server: %s", resp.Reply)
}