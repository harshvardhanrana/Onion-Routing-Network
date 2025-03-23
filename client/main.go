package main

import (
	"context"
	// "fmt"
	"log"
	routingpb "onion_routing/protofiles"
	utils "onion_routing/utils"
	"google.golang.org/grpc"
)

const (
	serverAddr = "localhost:23455"
	relayAddr = "localhost:34502"
)

var (
	clientLogger *utils.Logger
)

func main(){
	creds := utils.LoadCredentialsAsClient("certificates/ca.crt", 
											"certificates/client.crt",
											"certificates/client.key")

	clientLogger = utils.NewLogger("logs/client")
	conn, err := grpc.NewClient(relayAddr, grpc.WithTransportCredentials(creds))
	
	if err != nil {
		log.Fatalf("error while connecting to server: %v\n", err)
	}
	defer conn.Close()

	client := routingpb.NewRelayNodeServerClient(conn)
	req := &routingpb.DummyRequest{Message: "Hi, This is Client"}

	clientLogger.PrintLog("Request sending to server: %v", req)
	resp, err := client.RelayNodeRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("error whiling calling rpc: %v\n", err)
	}

	clientLogger.PrintLog("Response received from server: %v", resp)
	log.Printf("Response received from server: %s", resp.Reply)
}