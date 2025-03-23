PROTO_DIR = ./protofiles
CLIENT_DIR = ./client 
SERVER_DIR = ./server 
RELAY_NODE_DIR = ./relay 
DIRECTORY_SERVER_DIR = ./directory
LOGS_DIR = ./logs

PROTO_FILES = $(PROTO_DIR)/routing.proto 
PROTO_OUT_DIR = .

PROTO_COMPILE_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
					  --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative 

CLIENT_FILES = $(wildcard $(CLIENT_DIR)/*.go)
RELAY_NODE_FILES = $(wildcard $(RELAY_NODE_DIR)/*.go)
SERVER_FILES = $(wildcard $(SERVER_DIR)/*.go)
DIRECTORY_SERVER_FILES = $(wildcard $(DIRECTORY_SERVER_DIR)/*.go)

.PHONY: proto relay client server directory

proto:
	protoc $(PROTO_COMPILE_FLAGS) $(PROTO_FILES)

relay:
	go run $(RELAY_NODE_FILES)

client:
	go run $(CLIENT_FILES)

server:
	go run $(SERVER_FILES)

directory:
	go run $(DIRECTORY_SERVER_FILES)

clean_logs:
	rm -rf $(LOGS_DIR)/*

