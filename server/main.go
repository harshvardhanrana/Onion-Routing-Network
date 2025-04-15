package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

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
	resp := &routingpb.DummyResponse{Reply: []byte("Hello Client, I am Server")}
	serverLogger.PrintLog("Response received from next Node: %v", resp)
	return resp, nil
}

func (s *TestServer) TestRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error){
	message := req.Message
	serverLogger.PrintLog("Request received from client: %v", req)
	log.Printf("Message Received from client: %s\n", message)
	resp := &routingpb.DummyResponse{Reply: []byte("Hi, This is Test Server")}
	serverLogger.PrintLog("Response sending from server : %v", resp)
	return resp, nil
}

func fib(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	}
	return fib(n-1) + fib(n-2)
}

func (s *TestServer) Test1RPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error){
	message, err := strconv.Atoi(string(req.Message))
	if err != nil {
		return &routingpb.DummyResponse{}, err
	}
	serverLogger.PrintLog("Request received from client for fib: %v", req)
	log.Printf("Fib Request from client: %v\n", message)
	ret := fib(message)
	retString := fmt.Sprintf("Fibonacci of %v is %v", message, ret)
	return &routingpb.DummyResponse{Reply: []byte(retString)}, nil
}

func (s *TestServer) Test2RPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error) {
	message := string(req.Message)
	serverLogger.PrintLog("Request received from client: %v", req)
	log.Printf("Message Received from client: %s\n", message)
	retString := fmt.Sprintf("Welcome %s", message)
	resp := &routingpb.DummyResponse{Reply: []byte(retString)}
	serverLogger.PrintLog("Response sending from server : %v", resp)
	return resp, nil
}

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

	// server := grpc.NewServer(grpc.Creds(creds))
	// routingpb.RegisterRelayNodeServerServer(server, &RelayNodeServer{})
	// log.Printf("Server running on %s\n", utils.ServerAddr)
	// err = server.Serve(listener)
	// if err != nil {
	// 	log.Fatalf("server failed to server: %v", err)
	// }
	server := grpc.NewServer(grpc.Creds(creds))
	routingpb.RegisterTestServiceServer(server, &TestServer{})
	log.Printf("Test Server running on %s\n", utils.ServerAddr)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Test server failed to server: %v", err)
	}
}