package main  


import (
	"context"
	// "encoding/base64"
	"encoding/json"
	// "fmt"
	"log"
	"time"
	"crypto/rsa"

	// ecies "github.com/ecies/go/v2"

	utils "onion_routing/utils"

	"go.etcd.io/etcd/client/v3"
)

type RelayNode struct {
	Address string           `json:"address"`
	PubKey  *rsa.PublicKey `json:"pub_key"`
	Load    int              `json:"load"`
}


// getAvailableRelayNodes unmarshals relay node data from etcd.
// The custom UnmarshalJSON on RelayNode will automatically decode the public key.
func getAvailableRelayNodes(etcdClient *clientv3.Client) ([]RelayNode, error) {
	nodes = []RelayNode{}
	resp, err := etcdClient.Get(context.Background(), utils.EtcdKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Printf("Failed to fetch relay nodes: %v", err)
		return nodes, err
	}
	for _, ev := range resp.Kvs {
		var node RelayNode 
		err := json.Unmarshal(ev.Value, &node)
		if err != nil {
			log.Printf("Failed to decode relay node data: %v", err)
			continue
		}
		nodes = append(nodes, node)
	}
	log.Printf("Nodes: %v", nodes)
	return nodes, nil
}

func initEtcdClient() (*clientv3.Client, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{utils.EtcdServerAddr},
		DialTimeout: utils.EtcdTimeOutInterval * time.Second,
	})
	return etcdClient, err
}

func checkEtcdStatus(etcdClient *clientv3.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := etcdClient.Status(ctx, utils.EtcdServerAddr)
	return err
}