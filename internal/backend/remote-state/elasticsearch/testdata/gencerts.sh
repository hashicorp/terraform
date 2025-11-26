#!/usr/bin/env bash
#
# Generates certs required for elasticsearch testing:
# - ca.key and ca.cert.pem are self-signed, used as the source of truth for client and server to verify each other.
# - client.key and client.crt are the client's key and cert (signed by the ca key and cert)
# - server.key and server.crt are the server's key and cert (signed by the ca key and cert)

set -ex

# Nuke and recreate the certs dir
rm -rf certs
mkdir certs
cd certs || exit 1

# CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365000 -key ca.key -out ca.cert.pem -subj "/C=US/ST=State/L=City/O=Organization/OU=OU/CN=CN"

# Server
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr -addext 'subjectAltName = DNS:localhost' -subj "/C=US/ST=State/L=City/O=Organization/OU=OU/CN=CN"
openssl x509 -req -days 365000 -in server.csr -CA ca.cert.pem -CAkey ca.key -CAcreateserial -out server.crt -copy_extensions copy

# Client
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr -addext 'subjectAltName = DNS:localhost' -subj "/C=US/ST=State/L=City/O=Organization/OU=OU/CN=CN"
openssl x509 -req -days 365000 -in client.csr -CA ca.cert.pem -CAkey ca.key -CAcreateserial -out client.crt -copy_extensions copy
