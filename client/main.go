package main

import (
	"context"
	// "fmt"
	"log"

	routingpb "onion_routing/protofiles"
	logging "onion_routing/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddr = "localhost:23455"
)

var (
	clientLogger *logging.Logger
)

func main(){
	clientLogger = logging.NewLogger("logs/client")
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()
	client := routingpb.NewTestServiceClient(conn)
	req := &routingpb.DummyRequest{Message: "Hi, This is Client"}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.TestRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)
}