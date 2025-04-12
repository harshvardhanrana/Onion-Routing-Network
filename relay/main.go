package main

import (
	// "context"
	// "fmt"
	"context"
	"log"
	"net"
	"os"
	"fmt"
	"strconv"
	"google.golang.org/grpc"
	// "crypto/rsa"
	"google.golang.org/grpc/peer"
	// "google.golang.org/protobuf/proto"
	ecies "github.com/ecies/go/v2"
	
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"
	encryption "onion_routing/encryption"

	"google.golang.org/grpc/credentials"
	// "google.golang.org/grpc/metadata"
)

const (
	serverAddr = "localhost:23455"
)

type RelayNode struct {
	Address string `json:"address"`
	PubKey  string `json:"pub_key"`
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
}


var (
	relayLogger *utils.Logger
	relayCredsAsServer credentials.TransportCredentials
	relayCredsAsClient credentials.TransportCredentials
	relayAddr string
	nodeID string 
	// pubKey *rsa.PublicKey
	// privateKey *rsa.PrivateKey
	privateKey *ecies.PrivateKey
	pubKey *ecies.PublicKey
	load int
	circuitInfoMap = make(map[uint16]CircuitInfo)	// map of circuit id to circuit info
)

type RelayNodeServer struct {
	routingpb.UnimplementedRelayNodeServerServer
}

func handleCreateCell(cell encryption.OnionCell, ctx context.Context) uint16 {
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

	cinfo := CircuitInfo{
		BackF: cell.BackF,
		ForwF: cell.ForwF,
		Expiration: cell.Expiration,
		KeySeed: cell.KeySeed,
		ForwardIP: cell.IP,
		ForwardPort: cell.Port,
		BackwardIP: backIPBytes,
		BackwardPort: backPortUint16,
	}

	circuitInfoMap[cell.CircuitID] = cinfo

	return cell.CircuitID
}

func handleRequest(ctx context.Context, req *routingpb.DummyRequest) uint16 {
	// decryptedMessage, err := encryption.DecryptRSA(req.Message, privateKey)
	decryptedMessage, err := encryption.DecryptECC(req.Message, privateKey)
	if err != nil {
		log.Fatalf("Failed to decrypt message: %v", err)
	}

	fmt.Println("Decrypted message: ")
	rebuiltCell := encryption.RebuildMessage(decryptedMessage)
	fmt.Println(rebuiltCell.String())
	fmt.Println("Size of decrypted message: ", len(decryptedMessage))
	switch rebuiltCell.CellType {
	case 1: // create cell
		fmt.Println("Create cell")
		circuitID := handleCreateCell(rebuiltCell, ctx)
		return circuitID
	}
	return 0
}

func (s *RelayNodeServer) RelayNodeRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error) {
	relayLogger.PrintLog("Request recieved from client: %v", req)

	circuitID := handleRequest(ctx, req)
	cinfo, exists := circuitInfoMap[circuitID]
	if !exists {
		log.Fatalf("Circuit ID %d not found in circuitInfoMap", circuitID)
	}

	sendAddr := fmt.Sprintf("%d.%d.%d.%d:%d", 
		cinfo.ForwardIP[0], cinfo.ForwardIP[1], cinfo.ForwardIP[2], cinfo.ForwardIP[3], cinfo.ForwardPort)
	fmt.Println("Send Address: ", sendAddr)
 
	conn, err := grpc.NewClient(sendAddr, grpc.WithTransportCredentials(relayCredsAsClient))
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
	privateKey, pubKey, _ = genECCKeyPair()

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
	
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Relay Node server failed to server: %v", err)
	}
}