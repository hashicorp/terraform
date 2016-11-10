#!/bin/bash

set -ex

DOCKER_VERSION="1.10.3"

sudo stop docker
sudo rm -rf /var/lib/docker
sudo rm -f `which docker`
sudo apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
echo "deb https://apt.dockerproject.org/repo ubuntu-trusty main" | sudo tee /etc/apt/sources.list.d/docker.list
sudo apt-get update
sudo apt-get install -y --force-yes -o Dpkg::Options::="--force-confnew" docker-engine=$DOCKER_VERSION-0~`lsb_release -cs`

docker version
