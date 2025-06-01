# Onion Routing Network

This project implements a distributed routing system with components for clients, servers, relay nodes, and a directory service, all written in Go and using gRPC for communication.

## Directory Structure

* `protofiles/` – Contains `.proto` files defining gRPC services and messages.
* `client/` – Client implementation.
* `server/` – Server implementation.
* `relay/` – Relay node logic.
* `directory/` – Directory server.
* `logs/` – Runtime logs.

## Prerequisites

* Go (>=1.16)
* `protoc` (Protocol Buffers compiler)
* `protoc-gen-go` and `protoc-gen-go-grpc` plugins
* [etcd](https://etcd.io/) (for directory service)

## Makefile Targets

### `make proto`

Generates Go code from the protobuf file (`protofiles/routing.proto`).

### `make etcd`

Starts an `etcd` server locally on port `2379`.

### `make relay`

Runs a relay node. Set `RELAY_NODE_ID` if needed (default is `1`):

```sh
make relay RELAY_NODE_ID=2
```

### `make client`

Runs a client. Set `CLIENT_ID` if needed (default is `1001`):

```sh
make client CLIENT_ID=2001
```

### `make server`

Runs the server component.

### `make directory`

Starts the directory server.

### `make clean_logs`

Removes all logs from the `logs/` directory.

## Notes

* Ensure `etcd` is running before starting the directory server.
* All components communicate via gRPC using code generated from `routing.proto`.
