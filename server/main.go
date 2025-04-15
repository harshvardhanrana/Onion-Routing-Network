package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"math/rand/v2"

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

type OnionRoutingServer struct {
	routingpb.UnimplementedOnionRoutingServerServer
}

type RelayNodeServer struct {
	routingpb.UnimplementedRelayNodeServerServer
}

// func (s *RelayNodeServer) RelayNodeRPC(ctx context.Context, req *routingpb.RelayRequest) (*routingpb.RelayResponse, error) {
// 	serverLogger.PrintLog("Request recieved from previous Node: %v", req)
// 	resp := &routingpb.RelayResponse{Reply: []byte("Hello Client, I am Server")}
// 	serverLogger.PrintLog("Response received from next Node: %v", resp)
// 	return resp, nil
// }

func (s *OnionRoutingServer) GreetServer(ctx context.Context, req *routingpb.GreetRequest) (*routingpb.GreetResponse, error){
	message := req.Message
	serverLogger.PrintLog("Request received from client: %v", req)
	log.Printf("Message Received from client: %s\n", message)
	resp := &routingpb.GreetResponse{Reply: []byte("Welcome, This is Onion-Routing Server")}
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

func nRandomNumbers(n int) []int {
	ranNums := make([]int, n)
	for i := 0 ; i < n ; i++ {
		ranNums[i] = rand.IntN(100)
	}
	return ranNums
}

func (s *OnionRoutingServer) CalculateFibonacci(ctx context.Context, req *routingpb.FibonacciRequest) (*routingpb.FibonacciResponse, error){
	message, err := strconv.Atoi(string(req.N))
	if err != nil {
		return &routingpb.FibonacciResponse{}, err
	}
	serverLogger.PrintLog("Request received from client for fib: %v", req)
	log.Printf("Fib Request from client: %v\n", message)
	ret := fib(message)
	retString := fmt.Sprintf("Fibonacci of %v is %v", message, ret)
	return &routingpb.FibonacciResponse{Reply: []byte(retString)}, nil
}

func (s *OnionRoutingServer) GetRandomNumbers(ctx context.Context, req *routingpb.GetRandomRequest) (*routingpb.GetRandomResponse, error) {
	message, err := strconv.Atoi(string(req.N))
	if err != nil {
		return &routingpb.GetRandomResponse{}, err
	}
	serverLogger.PrintLog("Request received from client: %v", req)
	log.Printf("N Received from client: %d\n", message)
	randomNumbers := nRandomNumbers(message)
	rndNums := ""
	for i := 0 ; i < len(randomNumbers) ; i++ {
		rndNums += strconv.Itoa(randomNumbers[i])
		rndNums += ", "
	}
	retString := fmt.Sprintf("N-Random Numbers: %s", rndNums)
	resp := &routingpb.GetRandomResponse{Reply: []byte(retString)}
	serverLogger.PrintLog("Response sending from server : %v", resp)
	return resp, nil
}

func main() {
	serverAddr := flag.Int("port", 45034, "port number")
	flag.Parse()
	realAddr := fmt.Sprintf("localhost:%v", *serverAddr)
	creds := utils.LoadCredentialsAsServer("certificates/ca.crt", 
										"certificates/server.crt", 
										"certificates/server.key")	

	serverLogger = utils.NewLogger("logs/server")
	listener, err := net.Listen("tcp", realAddr)
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
	routingpb.RegisterOnionRoutingServerServer(server, &OnionRoutingServer{})
	log.Printf("Test Server running on %s\n", utils.ServerAddr)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Test server failed to server: %v", err)
	}
}