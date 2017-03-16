#!/bin/bash

set -eu
set -x

# Test domains
export DNS_DOMAIN_FORWARD="example.com."
export DNS_DOMAIN_REVERSE="1.168.192.in-addr.arpa."

# Run with no authentication

export DNS_UPDATE_SERVER=127.0.0.1
docker run -d -p 53:53/udp \
	-e BIND_DOMAIN_FORWARD=${DNS_DOMAIN_FORWARD} \
	-e BIND_DOMAIN_REVERSE=${DNS_DOMAIN_REVERSE} \
	-e BIND_INSECURE=true \
	--name bind_insecure drebes/bind
make testacc TEST=./builtin/providers/dns
docker stop bind_insecure
docker rm bind_insecure

# Run with authentication

export DNS_UPDATE_KEYNAME=${DNS_DOMAIN_FORWARD}
export DNS_UPDATE_KEYALGORITHM="hmac-md5"
export DNS_UPDATE_KEYSECRET="c3VwZXJzZWNyZXQ="
docker run -d -p 53:53/udp \
	-e BIND_DOMAIN_FORWARD=${DNS_DOMAIN_FORWARD} \
	-e BIND_DOMAIN_REVERSE=${DNS_DOMAIN_REVERSE} \
	-e BIND_KEY_NAME=${DNS_UPDATE_KEYNAME} \
	-e BIND_KEY_ALGORITHM=${DNS_UPDATE_KEYALGORITHM} \
	-e BIND_KEY_SECRET=${DNS_UPDATE_KEYSECRET} \
	--name bind_secure drebes/bind
make testacc TEST=./builtin/providers/dns
docker stop bind_secure
docker rm bind_secure
