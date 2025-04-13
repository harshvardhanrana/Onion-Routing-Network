package main

import (
	"context"
	"log"
	"net"
	// "os"
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	// "crypto/tls"
	// "crypto/x509"
)

var (
	serverLogger *utils.Logger
)

type TestServer struct {
	routingpb.UnimplementedTestServiceServer
}

type RelayNodeServer struct {
	routingpb.UnimplementedRelayNodeServerServer
}

func (s *RelayNodeServer) RelayNodeRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error) {
	serverLogger.PrintLog("Request recieved from previous Node: %v", req)
	resp := &routingpb.DummyResponse{Reply: "Hello Client, I am Server"}
	serverLogger.PrintLog("Response received from next Node: %v", resp)
	return resp, nil
}

// func (s *TestServer) TestRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error){
// 	message := req.Message
// 	serverLogger.PrintLog("Request received from client: %v", req)
// 	log.Printf("Message Received from client: %s\n", message)
// 	resp := &routingpb.DummyResponse{Reply: "Hi, This is Test Server"}
// 	serverLogger.PrintLog("Response sending from server : %v", resp)
// 	return resp, nil
// }

func main() {
	creds := utils.LoadCredentialsAsServer("certificates/ca.crt", 
										"certificates/server.crt", 
										"certificates/server.key")	

	serverLogger = utils.NewLogger("logs/server")
	listener, err := net.Listen("tcp", utils.ServerAddr)
	if err != nil {
		log.Fatalf("server failed to listen: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer(grpc.Creds(creds))
	routingpb.RegisterRelayNodeServerServer(server, &RelayNodeServer{})
	log.Printf("Server running on %s\n", utils.ServerAddr)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("server failed to server: %v", err)
	}
	// server := grpc.NewServer(grpc.Creds(creds))
	// routingpb.RegisterTestServiceServer(server, &TestServer{})
	// log.Printf("Test Server running on %s\n", serverAddr)
	// err = server.Serve(listener)
	// if err != nil {
	// 	log.Fatalf("Test server failed to server: %v", err)
	// }
}