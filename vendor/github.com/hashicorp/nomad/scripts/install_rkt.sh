#!/bin/bash

set -ex

RKT_VERSION="v1.5.1"
RKT_SHA512="8163ca59fc8c44c9c2997431d16274d81d2e82ff2956c860607f4c111de744b78cdce716f8afbacf7173e0cdce25deac73ec95a30a8849bbf58d35faeb84e398"
DEST_DIR="/usr/local/bin"

sudo mkdir -p /etc/rkt/net.d
echo '{"name": "default", "type": "ptp", "ipMasq": false, "ipam": { "type": "host-local", "subnet": "172.16.28.0/24", "routes": [ { "dst": "0.0.0.0/0" } ] } }' | sudo tee -a /etc/rkt/net.d/99-network.conf

if [ ! -d "rkt-${RKT_VERSION}" ]; then
    printf "rkt-%s/ doesn't exist\n" "${RKT_VERSION}"
    if [ ! -f "rkt-${RKT_VERSION}.tar.gz" ]; then
        printf "Fetching rkt-%s.tar.gz\n" "${RKT_VERSION}"
        wget https://github.com/coreos/rkt/releases/download/$RKT_VERSION/rkt-$RKT_VERSION.tar.gz
        expected_version=$(printf 'SHA512(rkt-%s.tar.gz)= %s' "${RKT_VERSION}" "${RKT_SHA512}")
        actual_version=$(openssl sha512 rkt-${RKT_VERSION}.tar.gz)
        if [ "${expected_version}" != "${actual_version}" ]; then
            printf "SHA512 of rkt-%s failed\n" "${RKT_VERSION}"
            exit 1
        fi
        tar xzvf rkt-$RKT_VERSION.tar.gz
    fi
fi

sudo cp rkt-$RKT_VERSION/rkt $DEST_DIR
sudo cp rkt-$RKT_VERSION/*.aci $DEST_DIR

rkt version
