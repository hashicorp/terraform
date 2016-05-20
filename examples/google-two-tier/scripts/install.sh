#!/bin/bash -xe

RESOURCE_INDEX=$1
apt-get -y update
apt-get -y install nginx
IP=$(curl -s -H "Metadata-Flavor:Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
echo "Welcome to Resource ${RESOURCE_INDEX} - ${HOSTNAME} (${IP})" > /usr/share/nginx/html/index.html
service nginx start
