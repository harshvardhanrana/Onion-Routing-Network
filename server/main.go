package main

import (
	"context"
	"log"
	"net"
	routingpb "onion_routing/protofiles"
	logging "onion_routing/common"
	"google.golang.org/grpc"
)


const (
	serverAddr = "localhost:23455"
)

var (
	serverLogger *logging.Logger
)

type TestServer struct {
	routingpb.UnimplementedTestServiceServer
}

func (s *TestServer) TestRPC(ctx context.Context, req *routingpb.DummyRequest) (*routingpb.DummyResponse, error){
	message := req.Message
	serverLogger.PrintLog("Request received from client: %v", req)
	log.Printf("Message Received from client: %s\n", message)
	resp := &routingpb.DummyResponse{Reply: "Hi, This is Test Server"}
	serverLogger.PrintLog("Response sending from server : %v", resp)
	return resp, nil
}

func main() {
	serverLogger = logging.NewLogger("logs/server")
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("server failed to listen: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	routingpb.RegisterTestServiceServer(server, &TestServer{})
	log.Printf("Test Server running on %s\n", serverAddr)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Test server failed to server: %v", err)
	}
}