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
	"math/rand"
	"strconv"
	"google.golang.org/grpc"
	// "crypto/rsa"
	"google.golang.org/grpc/peer"
	"go.etcd.io/etcd/client/v3"
	// "google.golang.org/protobuf/proto"
	// ecies "github.com/ecies/go/v2"
	"crypto/rsa"
	
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"
	encryption "onion_routing/encryption"

	"google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/metadata"
)

// const (
// 	serverAddr = "localhost:23455"
// )

type RelayNode struct {
	Address string `json:"address"`
	PubKey *rsa.PublicKey `json:"pub_key"`
	Load int `json:"load"`
}

// cell := OnionCell{
// 	CellType:   1,                    // Create cell
// 	CircuitID:  1001,                 // Example Circuit ID
// 	Version:    1,                    // Version 1
// 	BackF:      1,                    // Backward cipher (e.g., DES)
// 	ForwF:      2,                    // Forward cipher (e.g., RC4)
// 	Port:       9002,                 // Port number
// 	IP:         [4]byte{192, 168, 1, 1}, // Destination IP
// 	Expiration: 1700000000,           // Expiration time
// 	KeySeed:    [16]byte{'1', '6', 'B', 'y', 't', 'e', 's', 'K', 'e', 'y', 'S', 'e', 'e', 'd', '!'},
// 	Payload:    []byte("Hello, Onion!"), // Payload
// }

type CircuitInfo struct {
	BackF byte
	ForwF byte
	ForwardIP [4]byte
	ForwardPort uint16
	BackwardIP [4]byte
	BackwardPort uint16
	Expiration uint32
	KeySeed [16]byte
	key1 [8]byte 
	key2 [16]byte
	key3 [16]byte
}


var (
	relayLogger *utils.Logger
	relayCredsAsServer credentials.TransportCredentials
	relayCredsAsClient credentials.TransportCredentials
	relayAddr string
	nodeID string 
	// pubKey *rsa.PublicKey
	// privateKey *rsa.PrivateKey
	privateKey *rsa.PrivateKey
	pubKey *rsa.PublicKey
	load int
	circuitInfoMap = make(map[uint16]CircuitInfo)	// map of circuit id to circuit info
)

type RelayNodeServer struct {
	routingpb.UnimplementedRelayNodeServerServer
}

func handleCreateCell(cell encryption.OnionCell, ctx context.Context)(CircuitInfo){
	p, ok := peer.FromContext(ctx)
	if !ok {
		log.Println("Could not extract peer from context")
	} else {
		relayLogger.PrintLog("Request received from: %v", p.Addr.String())
	}

	_, backPort := utils.GetPortAndIP(p.Addr.String())
	//TODO: Temporarily making backIP same as forward IP as localhost
	backIPBytes := cell.IP
	backPortUint, _ := strconv.Atoi(string(backPort[:]))
	backPortUint16 := uint16(backPortUint)
	key1, key2, key3 := encryption.DeriveKeys(cell.KeySeed[:])

	cinfo := CircuitInfo{
		BackF: cell.BackF,
		ForwF: cell.ForwF,
		Expiration: cell.Expiration,
		KeySeed: cell.KeySeed,
		ForwardIP: cell.IP,
		ForwardPort: cell.Port,
		BackwardIP: backIPBytes,
		BackwardPort: backPortUint16,
		key1: [8]byte(key1),
		key2: [16]byte(key2),
		key3: [16]byte(key3),
	}

	circuitInfoMap[cell.CircuitID] = cinfo
	return cinfo
}

// func decryptMessageHeader(messageHeader []byte, privateKey *rsa.PrivateKey){
// 	decryptedHeader, err := encryption.DecryptRSA(messageHeader, privateKey)
// 	return 
// }

func handleRequest(ctx context.Context, req *routingpb.DummyRequest) (CircuitInfo, []byte){
	// decryptedMessage, err := encryption.DecryptRSA(req.Message, privateKey)
	encryptedMessageHeader := req.Message[:256]
	encryptedMessagePayload := req.Message[256:]
	decryptedMessageHeader, err := encryption.DecryptRSA(encryptedMessageHeader, privateKey)
	if err != nil {
		log.Fatalf("Failed to decrypt message: %v", err)
	}

	// log.Println("Decrypted message: ")
	rebuiltCell := encryption.RebuildMessage(decryptedMessageHeader)
	// log.Println(rebuiltCell.String()) 
	// log.Println("Size of decrypted message: ", len(decryptedMessageHeader))
	switch rebuiltCell.CellType {
	case 1: // create cell
		log.Println("Create cell")
		circuitInfo := handleCreateCell(rebuiltCell, ctx)
		circuitInfoMap[rebuiltCell.CircuitID] = circuitInfo
		return circuitInfo, encryptedMessagePayload
	case 2:
		log.Println("Data cell")
		cinfo, exists := circuitInfoMap[rebuiltCell.CircuitID]
		if !exists {
			log.Fatalf("Circuit ID %d not found in circuitInfoMap", rebuiltCell.CircuitID)
		}
		decryptedMessagePayload := encryption.DecryptRC4(encryptedMessagePayload, cinfo.key1[:])
		// log.Println("Decrypted message payload: ", string(decryptedMessagePayload))
		return cinfo, decryptedMessagePayload
	case 4:
		log.Println("Padding cell")
		break
	}
	return CircuitInfo{}, make([]byte, 0)
}

func handleResponse(circuitInfo CircuitInfo, respMessage []byte) ([]byte){
	encryptedRespMessage := encryption.EncryptRC4(respMessage, circuitInfo.key2[:])
	return encryptedRespMessage
}	

func (s *RelayNodeServer) RelayNodeRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error) {
	relayLogger.PrintLog("Request recieved from previous Node: %v", req)

	circuitInfo, forwardMessage := handleRequest(ctx, req)
	nextNodeAddr := fmt.Sprintf("localhost:%d",circuitInfo.ForwardPort)
	log.Println("Sending to Node with Addr: ", nextNodeAddr)

	if len(forwardMessage) == 0 {
		return &routingpb.DummyResponse{Reply: []byte("No forward message")}, nil
	}
 
	conn, err := grpc.NewClient(nextNodeAddr, grpc.WithTransportCredentials(relayCredsAsClient))
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()
	client := routingpb.NewRelayNodeServerClient(conn)

	forwardReq := &routingpb.DummyRequest{Message: forwardMessage}
	relayLogger.PrintLog("Request sending to next Node: %v", forwardReq)
	resp, err := client.RelayNodeRPC(context.Background(), forwardReq)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}
	relayLogger.PrintLog("Response received from next Node: %v", resp)
	// return resp, nil
	respMessage := handleResponse(circuitInfo, []byte(resp.Reply))
	backwardResp := &routingpb.DummyResponse{Reply: respMessage}
	return backwardResp, nil
}

func encryptPaddingMessage(message []byte, pubkey *rsa.PublicKey)([]byte, error){
	messageHeader := message[:32]
	messagePayload := message[32:]
	
	encryptedHeader, err := encryption.EncryptRSA(messageHeader, pubkey)
	if err != nil {
		return make([]byte, 0), err
	}
	encryptedMessage := append(encryptedHeader, messagePayload...)
	
	return encryptedMessage, nil
}

func paddingLoopRandom(etcdClient *clientv3.Client, creds credentials.TransportCredentials, selfAddr string) {
	count := 1
	for {
		nodes, err := GetAvailableRelayNodes(etcdClient)
		if err != nil || len(nodes) == 0 {
			log.Println("No available nodes for padding.")
			time.Sleep(2 * time.Second)
			continue
		}

		others := make([]RelayNode, 0)
		for _, n := range nodes {
			if n.Address != selfAddr {
				others = append(others, n)
			}
		}
		if len(others) == 0 {
			log.Println("Only this relay is registered; skipping padding.")
			time.Sleep(2 * time.Second)
			continue
		}

		target := others[rand.Intn(len(others))]

		cell := encryption.PaddingCell(count)
		message := encryption.BuildMessage(cell)
		encryptedMessage, err := encryptPaddingMessage(message, target.PubKey)

		conn, err := grpc.NewClient(target.Address, grpc.WithTransportCredentials(relayCredsAsClient))
		if err != nil {
			log.Printf("Padding failed: could not connect to %s: %v", target.Address, err)
			time.Sleep(1 * time.Second)
			continue
		}
		client := routingpb.NewRelayNodeServerClient(conn)

		_, err = client.RelayNodeRPC(context.Background(), &routingpb.DummyRequest{Message: encryptedMessage})
		if err != nil {
			log.Printf("Padding failed to %s: %v", target.Address, err)
		} 
		// else {
		// 	log.Printf("Sent padding to: %s", target.Address)
		// }
		conn.Close()

		// Wait random delay before next padding
		time.Sleep(time.Duration(rand.Intn(5000)+5000) * time.Millisecond)
		count++
	}
}

func main(){
	args := os.Args[1:]
	if len(args) >= 1 {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("Invalid command line argument; expecting integer value")
		}
		nodeID = fmt.Sprintf("node%d",id)
		// pubKey = fmt.Sprintf("node%d_pub_key",id)
	}

	// RSA
	// privateKey, pubKey = genKeyPairs()

	//ECC

	privateKey, pubKey = genKeyPairs()

	// privateKeyBytes := encodePrivateKeyToPEM(privateKey)
	// pubKeyBytes := encodePublicKeyToPEM(pubKey) 

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

	go paddingLoopRandom(etcdClient, relayCredsAsClient, relayAddr)
	
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Relay Node server failed to server: %v", err)
	}
}