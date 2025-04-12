package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	// "crypto/rsa"
	"strings"
	"strconv"
	ecies "github.com/ecies/go/v2"

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
	// PubKey *rsa.PublicKey `json:"pub_key"`
	PubKey *ecies.PublicKey `json:"pub_key"`
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

func getPortAndIP(address string) (uint16, [4]byte) {
	parts := strings.Split(address, ":")
	port, _ := strconv.Atoi(parts[1])
	// ip := parts[0]
	// ipParts := strings.Split(ip, ".")
	// var ipBytes [4]byte
	// for i, part := range ipParts {
	// 	num, err := strconv.Atoi(part)
	// 	if err != nil {
	// 		log.Fatalf("Invalid IP address: %v", err)
	// 	}
	// 	ipBytes[i] = byte(num)
	// }
	ipBytes := [4]byte{192, 168, 1, 1}
	return uint16(port), ipBytes
}

func startCreationRoute(client routingpb.RelayNodeServerClient, chosen_nodes []RelayNode) {

	// Innermost Layer (node 3)
	third_port, third_ip := getPortAndIP(chosen_nodes[2].Address)
	third_cell := encryption.CreateCell(third_ip, third_port, []byte("Hello World"))
	var third_message = encryption.BuildMessage(third_cell)

	fmt.Printf("Built Message:\n%x\n", third_message)
	fmt.Printf("Size of third_message: %d bytes\n", len(third_message))

	// encrypted_third_message, _ := encryption.EncryptRSA(third_message, chosen_nodes[2].PubKey)
	encrypted_third_message, _ := encryption.EncryptECC(third_message, chosen_nodes[2].PubKey)
	// encrypted_third_message := third_message

	// Middle Layer (node 2)
	second_port, second_ip := getPortAndIP(chosen_nodes[1].Address)
	second_cell := encryption.CreateCell(second_ip, second_port, encrypted_third_message)
	var second_message = encryption.BuildMessage(second_cell)
	fmt.Printf("Built Message:\n%x\n", second_message)
	fmt.Printf("Size of second_message: %d bytes\n", len(second_message))

	// encrypted_second_message, _ := encryption.EncryptRSA(second_message, chosen_nodes[1].PubKey)
	encrypted_second_message, _ := encryption.EncryptECC(second_message, chosen_nodes[1].PubKey)
	// encrypted_second_message := second_message

	// Outermost Layer (node 1)
	first_port, first_ip := getPortAndIP(chosen_nodes[0].Address)
	first_cell := encryption.CreateCell(first_ip, first_port, encrypted_second_message)
	// fmt.Printf("Size of first_cell payload: %d bytes\n", len(first_cell.Payload))
	var first_message = encryption.BuildMessage(first_cell)
	fmt.Printf("Built Message:\n%x\n", first_message)
	fmt.Printf("Size of first_message: %d bytes\n", len(first_message))

	// encrypted_first_message, _ := encryption.EncryptRSA(first_message, chosen_nodes[0].PubKey)
	encrypted_first_message, _ := encryption.EncryptECC(first_message, chosen_nodes[0].PubKey)
	// encrypted_first_message := first_message

	// Send the encrypted message to the first relay node
	req := &routingpb.DummyRequest{Message: encrypted_first_message}

	// Client -> OR1
	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)

}

//TODO: Immplement this function later
func GetNodesInRoute(nodes []RelayNode) []RelayNode {
	chosen_nodes := []RelayNode{}
	for i := 0; i < 3; i++ {
		fmt.Println("IP Address: ", nodes[i].Address)
		chosen_nodes = append(chosen_nodes, nodes[i])
	}
	return chosen_nodes
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
	
	chosen_nodes := GetNodesInRoute(nodes)
	
	clientLogger = utils.NewLogger("logs/client")
	conn, err := grpc.NewClient(nodes[0].Address, grpc.WithTransportCredentials(creds))
	
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()

	client := routingpb.NewRelayNodeServerClient(conn)

	startCreationRoute(client, chosen_nodes)
	// req := &routingpb.DummyRequest{Message: "Hi, This is Client"}

	// clientLogger.PrintLog("Request sending to server: %v", req)
	// resp, err := client.RelayNodeRPC(context.Background(), req)
	// if err != nil {
	// 	log.Fatalf("error whiling calling rpc: %v\n", err)
	// }

	// clientLogger.PrintLog("Response received from server: %v", resp)
	// log.Printf("Response received from server: %s", resp.Reply)
}