package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	ecies "github.com/ecies/go/v2"

	routingpb "onion_routing/protofiles"
	encryption "onion_routing/encryption"
	utils "onion_routing/utils"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

var (
	clientLogger *utils.Logger
	nodes        []RelayNode
)

type RelayNode struct {
	Address string           `json:"address"`
	PubKey  *ecies.PublicKey `json:"pub_key"`
	Load    int              `json:"load"`
}

// Custom UnmarshalJSON method to handle decoding the pub_key
func (r *RelayNode) UnmarshalJSON(data []byte) error {
	// Create a temporary struct that stores the pub_key as a string.
	type Alias RelayNode
	aux := &struct {
		PubKey string `json:"pub_key"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// Unmarshal into the auxiliary struct.
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Decode the base64 encoded public key string.
	pubKeyBytes, err := base64.StdEncoding.DecodeString(aux.PubKey)
	if err != nil {
		return fmt.Errorf("failed to decode pub_key: %v", err)
	}

	// Use ecies.ImportECDSAPublic to import the decoded bytes.
	r.PubKey, err = ecies.NewPublicKeyFromBytes(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to import pub_key: %v", err)
	}

	return nil
}

// getAvailableRelayNodes unmarshals relay node data from etcd.
// The custom UnmarshalJSON on RelayNode will automatically decode the public key.
func getAvailableRelayNodes(etcdClient *clientv3.Client) ([]RelayNode, error) {
	nodes := []RelayNode{}
	resp, err := etcdClient.Get(context.Background(), utils.EtcdKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to fetch relay nodes: %v", err)
		return nodes, err
	}

	for _, ev := range resp.Kvs {
		var node RelayNode
		// This call uses the custom UnmarshalJSON method.
		if err := json.Unmarshal(ev.Value, &node); err != nil {
			log.Printf("Failed to decode relay node data: %v", err)
			continue
		}
		nodes = append(nodes, node)
	}

	log.Printf("Nodes: %v", nodes)
	return nodes, nil
}

func initEtcdClient() (*clientv3.Client, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{utils.EtcdServerAddr},
		DialTimeout: utils.EtcdTimeOutInterval * time.Second,
	})
	return etcdClient, err
}

func checkEtcdStatus(etcdClient *clientv3.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := etcdClient.Status(ctx, utils.EtcdServerAddr)
	return err
}

func getPortAndIP(address string) (uint16, [4]byte) {
	parts := strings.Split(address, ":")
	port, _ := strconv.Atoi(parts[1])
	ipBytes := [4]byte{192, 168, 1, 1}
	return uint16(port), ipBytes
}

func startCreationRoute(client routingpb.RelayNodeServerClient, chosen_nodes []RelayNode) {
	// Innermost Layer (node 3)
	server_port, server_ip := getPortAndIP(chosen_nodes[2].Address)
	server_port = uint16(23455)
	third_cell := encryption.CreateCell(server_ip, server_port, []byte("Hello World"))
	third_message := encryption.BuildMessage(third_cell)

	// Encrypt using ECC
	encrypted_third_message, err := encryption.EncryptECC(third_message, chosen_nodes[2].PubKey)
	if err != nil {
		log.Printf("Error encrypting third message: %v", err)
		return
	}

	// Middle Layer (node 2)
	third_port, third_ip := getPortAndIP(chosen_nodes[2].Address)
	second_cell := encryption.CreateCell(third_ip, third_port, encrypted_third_message)
	second_message := encryption.BuildMessage(second_cell)
	encrypted_second_message, err := encryption.EncryptECC(second_message, chosen_nodes[1].PubKey)
	if err != nil {
		log.Printf("Error encrypting second message: %v", err)
		return
	}

	// Outermost Layer (node 1)
	second_port, second_ip := getPortAndIP(chosen_nodes[1].Address)
	// first_port, first_ip := getPortAndIP(chosen_nodes[0].Address)
	first_cell := encryption.CreateCell(second_ip, second_port, encrypted_second_message)
	first_message := encryption.BuildMessage(first_cell)
	encrypted_first_message, err := encryption.EncryptECC(first_message, chosen_nodes[0].PubKey)
	if err != nil {
		log.Printf("Error encrypting first message: %v", err)
		return
	}

	req := &routingpb.DummyRequest{Message: encrypted_first_message}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error while calling rpc: %v\n", err)
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)
}

// GetNodesInRoute is a helper to pick 3 nodes from the list.
func GetNodesInRoute(nodes []RelayNode) []RelayNode {
	chosen_nodes := []RelayNode{}
	for i := 0; i < 3 && i < len(nodes); i++ {
		// fmt.Println("IP Address:", nodes[i].Address)
		chosen_nodes = append(chosen_nodes, nodes[i])
	}
	return chosen_nodes
}

func main() {
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

	nodes, err = getAvailableRelayNodes(etcdClient)
	if err != nil {
		log.Fatalf("Error while fetching available relays: %v", err)
	}

	chosen_nodes := GetNodesInRoute(nodes)

	clientLogger = utils.NewLogger("logs/client")
	// Use grpc.Dial to create a connection.
	conn, err := grpc.Dial(nodes[0].Address, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Error while connecting to server: %v\n", err)
	}
	defer conn.Close()

	client := routingpb.NewRelayNodeServerClient(conn)

	startCreationRoute(client, chosen_nodes)
}
