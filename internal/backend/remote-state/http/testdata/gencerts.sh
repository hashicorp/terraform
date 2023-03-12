#!/usr/bin/env bash
#
# Generates certs required for mTLS testing:
# - ca.key and ca.cert.pem are self-signed, used as the source of truth for client and server to verify each other.
# - client.key and client.crt are the client's key and cert (signed by the ca key and cert)
# - server.key and server.crt are the server's key and cert (signed by the ca key and cert)

set -ex

# I was doing this on M1 mac and needed newer openssl to add the SAN IP; please export OPENSSL when invoking as needed
OPENSSL="${OPENSSL:-openssl}"

# Nuke and recreate the certs dir
rm -rf certs
mkdir certs
cd certs || exit 1

# CA
"$OPENSSL" genrsa -out ca.key 4096
"$OPENSSL" req -new -x509 -days 365000 -key ca.key -out ca.cert.pem

# Server
"$OPENSSL" genrsa -out server.key 4096
"$OPENSSL" req -new -key server.key -out server.csr -addext 'subjectAltName = IP:127.0.0.1'
"$OPENSSL" x509 -req -days 365000 -in server.csr -CA ca.cert.pem -CAkey ca.key -CAcreateserial -out server.crt -copy_extensions copy

# Client
"$OPENSSL" genrsa -out client.key 4096
"$OPENSSL" req -new -key client.key -out client.csr -addext 'subjectAltName = IP:127.0.0.1'
"$OPENSSL" x509 -req -days 365000 -in client.csr -CA ca.cert.pem -CAkey ca.key -CAcreateserial -out client.crt -copy_extensions copy
