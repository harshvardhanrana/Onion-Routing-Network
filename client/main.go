package main

import (
	"context"
	"log"

	encryption "onion_routing/encryption"
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"

	"crypto/rand"
	"crypto/rsa"

	"google.golang.org/grpc"
)

var (
	clientLogger *utils.Logger
	nodes        []RelayNode
)



// func getPortAndIP(address string) (uint16, [4]byte) {
// 	parts := strings.Split(address, ":")
// 	port, _ := strconv.Atoi(parts[1])
// 	ipBytes := [4]byte{192, 168, 1, 1}
// 	return uint16(port), ipBytes
// }

func encryptCreateMessage(message []byte, pubkey *rsa.PublicKey)([]byte, error){
	messageHeader := message[:32]
	messagePayload := message[32:]
	
	encryptedHeader, err := encryption.EncryptRSA(messageHeader, pubkey)
	if err != nil {
		return make([]byte, 0), err
	}
	encryptedMessage := append(encryptedHeader, messagePayload...)
	log.Printf("Length of encrypted Message - Header:%d, PayLoad:%d", len(encryptedHeader), len(messagePayload))
	
	return encryptedMessage, nil
}

func encryptDataMessage(message []byte, pubkey *rsa.PublicKey, keySeed [16]byte)([]byte, error) {
	key1, _, _ := encryption.DeriveKeys(keySeed[:])
	messageHeader := message[:32]
	messagePayload := message[32:]
	encryptedHeader, err := encryption.EncryptRSA(messageHeader, pubkey)
	if err != nil {
		return make([]byte, 0), err
	}
	encryptedPayload := encryption.EncryptRC4(messagePayload, key1)
	encryptedMessage := append(encryptedHeader, encryptedPayload...)
	return encryptedMessage, nil
}

func buildLayer(cellType int, serverAddr string, circuitID uint16, keySeed [16]byte, pubkey *rsa.PublicKey, payload []byte)([]byte, error){
	server_port, server_ip := utils.GetPortAndIP(serverAddr)  // added server address
	var err error
	var encrypted []byte
	
	switch cellType {
	case 1:
		cell := encryption.CreateCell(server_ip, server_port, payload, circuitID, keySeed)
		message := encryption.BuildMessage(cell)
		encrypted, err = encryptCreateMessage(message, pubkey)
	case 2:
		cell := encryption.DataCell(payload, circuitID)
		message := encryption.BuildMessage(cell)
		encrypted, err = encryptDataMessage(message, pubkey, keySeed)
	}

	if err != nil {
		return make([]byte, 0), err
	}
	return encrypted, err
}

func startCreationRoute(client routingpb.RelayNodeServerClient, chosen_nodes []RelayNode, circuitID uint16, keySeed [16]byte)(error) {
	// Innermost Layer (node 3)

	encryptedMessage, err := buildLayer(1, utils.ServerAddr, circuitID, keySeed, chosen_nodes[2].PubKey, []byte("Create Cell Test"))
	if err != nil {
		return err
	}

	encryptedMessage, err = buildLayer(1, chosen_nodes[2].Address, circuitID, keySeed, chosen_nodes[1].PubKey, encryptedMessage)
	if err != nil {
		return err
	}

	encryptedMessage, err = buildLayer(1, chosen_nodes[1].Address, circuitID, keySeed, chosen_nodes[0].PubKey, encryptedMessage)
	if err != nil {
		return err
	}

	// // Middle Layer (node 2)
	// third_port, third_ip := utils.GetPortAndIP(chosen_nodes[2].Address)
	// second_cell := encryption.CreateCell(third_ip, third_port, encrypted_third_message, circuitID, keySeed)
	// second_message := encryption.BuildMessage(second_cell)
	// encrypted_second_message, err := encryption.EncryptECC(second_message, chosen_nodes[1].PubKey)
	// if err != nil {
	// 	log.Printf("Error encrypting second message: %v", err)
	// 	return
	// }

	// // Outermost Layer (node 1)
	// second_port, second_ip := utils.GetPortAndIP(chosen_nodes[1].Address)
	// // first_port, first_ip := getPortAndIP(chosen_nodes[0].Address)
	// first_cell := encryption.CreateCell(second_ip, second_port, encrypted_second_message, circuitID, keySeed)
	// first_message := encryption.BuildMessage(first_cell)
	// encrypted_first_message, err := encryption.EncryptECC(first_message, chosen_nodes[0].PubKey)
	// if err != nil {
	// 	log.Printf("Error encrypting first message: %v", err)
	// 	return
	// }

	req := &routingpb.DummyRequest{Message: encryptedMessage}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		return err
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)
	return nil
}

func sendRequest(client routingpb.RelayNodeServerClient, chosen_nodes []RelayNode, circuitID uint16, keySeed [16]byte, message string) (error) {

	encryptedMessage, err := buildLayer(2, utils.ServerAddr, circuitID, keySeed, chosen_nodes[2].PubKey, []byte(message))
	if err != nil {
		return err
	}

	encryptedMessage, err = buildLayer(2, chosen_nodes[2].Address, circuitID, keySeed, chosen_nodes[1].PubKey, encryptedMessage)
	if err != nil {
		return err
	}

	encryptedMessage, err = buildLayer(2, chosen_nodes[1].Address, circuitID, keySeed, chosen_nodes[0].PubKey, encryptedMessage)
	if err != nil {
		return err
	}

	// Innermost Layer (node 3)

	// third_cell := encryption.DataCell([]byte("Hi, This is request from client"), circuitID)
	// third_message := encryption.BuildMessage(third_cell)
	// encrypted_third_message := encryption.EncryptRC4(third_message, key1)

	// // Middle Layer (node 2)
	// second_cell := encryption.DataCell(encrypted_third_message, circuitID)
	// second_message := encryption.BuildMessage(second_cell)
	// encrypted_second_message := encryption.EncryptRC4(second_message, key1)

	// // Outermost Layer (node 1)
	// first_cell := encryption.DataCell(encrypted_second_message, circuitID)
	// first_message := encryption.BuildMessage(first_cell)
	// encrypted_first_message := encryption.EncryptRC4(first_message, key1)

	req := &routingpb.DummyRequest{Message: encryptedMessage}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error while calling rpc: %v\n", err)
	}
	clientLogger.PrintLog("Response received from server: %v", resp)

	log.Printf("Response received from server: %s", resp.Reply)
	return nil
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
	if len(nodes) < 3 {
		log.Fatalf("Insufficient Relay Nodes available");
	}
	chosen_nodes := GetNodesInRoute(nodes)

	clientLogger = utils.NewLogger("logs/client")
	// Use grpc.Dial to create a connection.
	conn, err := grpc.NewClient(nodes[0].Address, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Error while connecting to server: %v\n", err)
	}
	defer conn.Close()

	client := routingpb.NewRelayNodeServerClient(conn)

	key_seed := make([]byte, 16)
	rand.Read(key_seed)
	circuitID := 1001
	message := "This is Test Data Message"
	startCreationRoute(client, chosen_nodes, uint16(circuitID), [16]byte(key_seed))
	sendRequest(client, chosen_nodes, uint16(circuitID), [16]byte(key_seed), message)
}
