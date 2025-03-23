#!/bin/bash

# Exit immediately if any command fails
set -e

# Create a directory for certificates
CERTS_DIR="../certificates"
mkdir -p $CERTS_DIR

echo "Generating TLS Certificates in $CERTS_DIR..."

echo "Creating Certificate Authority (CA)..."
openssl genrsa -out $CERTS_DIR/ca.key 4096
openssl req -x509 -new -nodes -key $CERTS_DIR/ca.key -sha256 -days 365 -out $CERTS_DIR/ca.crt -subj "/CN=MyCA"

echo "Creating Relay Node Certificate..."
openssl genrsa -out $CERTS_DIR/relay_node.key 4096
openssl req -new -key $CERTS_DIR/relay_node.key -out $CERTS_DIR/relay_node.csr -subj "/CN=relay_node"

# Create a config file for SAN (Subject Alternative Name)
cat <<EOF > $CERTS_DIR/relay_node.ext
subjectAltName = DNS:relay_node, DNS:localhost, IP:127.0.0.1
EOF

openssl x509 -req -in $CERTS_DIR/relay_node.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
    -out $CERTS_DIR/relay_node.crt -days 365 -sha256 -extfile $CERTS_DIR/relay_node.ext

echo "Creating Server Certificate..."
openssl genrsa -out $CERTS_DIR/server.key 4096
openssl req -new -key $CERTS_DIR/server.key -out $CERTS_DIR/server.csr -subj "/CN=server"

# Create a config file for SAN (Subject Alternative Name)
cat <<EOF > $CERTS_DIR/server.ext
subjectAltName = DNS:server, DNS:localhost, IP:127.0.0.1
EOF

# Sign the Bank Server certificate with the CA
openssl x509 -req -in $CERTS_DIR/server.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
    -out $CERTS_DIR/server.crt -days 365 -sha256 -extfile $CERTS_DIR/server.ext

echo "🔹 Creating Client Certificate..."
openssl genrsa -out $CERTS_DIR/client.key 4096
openssl req -new -key $CERTS_DIR/client.key -out $CERTS_DIR/client.csr -subj "/CN=client"

# Sign the client certificate with the CA
openssl x509 -req -in $CERTS_DIR/client.csr -CA $CERTS_DIR/ca.crt -CAkey $CERTS_DIR/ca.key -CAcreateserial \
    -out $CERTS_DIR/client.crt -days 365 -sha256


echo "Certificates Generated Successfully!"
ls -l $CERTS_DIR
