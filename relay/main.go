package main

import (
	// "context"
	// "fmt"
	"context"
	"log"
	"net"
	
	"google.golang.org/grpc"
	
	"google.golang.org/grpc/credentials"
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"
	// "google.golang.org/grpc/metadata"
)

const (
	serverAddr = "localhost:23455"
	relayAddr = "localhost:34502"
)

var (
	relayLogger *utils.Logger
	relayCredsAsServer credentials.TransportCredentials
	relayCredsAsClient credentials.TransportCredentials
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



func main(){
	relayCredsAsClient = utils.LoadCredentialsAsClient("certificates/ca.crt", 
												  "certificates/relay_node.crt",
												  "certificates/relay_node.key")

	relayCredsAsServer = utils.LoadCredentialsAsServer("certificates/ca.crt", 
												  "certificates/relay_node.crt",
												  "certificates/relay_node.key")

	relayLogger = utils.NewLogger("logs/relay")
	listener, err := net.Listen("tcp", relayAddr)
	if err != nil {
		log.Fatalf("relay server failed to listen: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer(grpc.Creds(relayCredsAsServer))
	routingpb.RegisterRelayNodeServerServer(server, &RelayNodeServer{})
	log.Printf("Relay Node Server running on %s\n", serverAddr)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Relay Node server failed to server: %v", err)
	}
}