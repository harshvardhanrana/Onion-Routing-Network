package main

import (
	// "context"
	// "fmt"
	"context"
	"log"
	"net"
	"os"
	"fmt"
	"time"
	"strconv"
	"encoding/json"
	"google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"

	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/metadata"
)

const (
	serverAddr = "localhost:23455"
)

type RelayNode struct {
	Address string `json:"address"`
	PubKey string `json:"pub_key"`
	Load int `json:"load"`
}

var (
	relayLogger *utils.Logger
	relayCredsAsServer credentials.TransportCredentials
	relayCredsAsClient credentials.TransportCredentials
	relayAddr string
	nodeID string 
	pubKey string
	load int
)

type RelayNodeServer struct {
	routingpb.UnimplementedRelayNodeServerServer
}

func (s *RelayNodeServer) RelayNodeRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error) {
	relayLogger.PrintLog("Request recieved from client: %v", req)

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(relayCredsAsClient))
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()
	client := routingpb.NewTestServiceClient(conn)

	relayLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.TestRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}
	relayLogger.PrintLog("Response received from server: %v", resp)
	return resp, err
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


func main(){
	args := os.Args[1:]
	if len(args) >= 1 {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("Invalid command line argument; expecting integer value")
		}
		nodeID = fmt.Sprintf("node%d",id)
		pubKey = fmt.Sprintf("node%d_pub_key",id)
	}

	relayCredsAsClient = utils.LoadCredentialsAsClient("certificates/ca.crt", 
												  "certificates/relay_node.crt",
												  "certificates/relay_node.key")

	relayCredsAsServer = utils.LoadCredentialsAsServer("certificates/ca.crt", 
												  "certificates/relay_node.crt",
												  "certificates/relay_node.key")

    var err error
	relayLogger = utils.NewLogger("logs/relay")
	relayAddr, err = utils.GetAvaliableAddress()
	if err != nil {
		log.Fatalf("Failed to get server address: %v", err)
	}

	// server initialization 
	listener, err := net.Listen("tcp", relayAddr)
	if err != nil {
		log.Fatalf("relay server failed to listen: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer(grpc.Creds(relayCredsAsServer))
	routingpb.RegisterRelayNodeServerServer(server, &RelayNodeServer{})
	log.Printf("Relay Node Server running on %s\n", relayAddr)
	

	// etcd registration
	etcdClient, err := initEtcdClient()
	if err != nil {
		log.Fatalf("Failed to initialize etcd client: %v", err)
	}
	defer etcdClient.Close()
	err = checkEtcdStatus(etcdClient)
	if err != nil {
		log.Fatalf("Etcd Server is unreachable: %v", err)
	}
	leaseId, err := createLease(etcdClient)
	if err != nil {
		log.Fatalf("Failed to create Etcd lease: %v", err)
	}
	err = registerWithEtcdServer(etcdClient, leaseId)
	if err != nil {
		log.Fatalf("Failed to register with Etcd: %v", err)
	}
	go keepAliveThread(etcdClient, leaseId)
	
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Relay Node server failed to server: %v", err)
	}
}