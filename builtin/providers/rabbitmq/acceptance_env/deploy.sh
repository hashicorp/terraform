#!/bin/bash
set -e

cd
echo 'deb http://www.rabbitmq.com/debian/ testing main' | sudo tee /etc/apt/sources.list.d/rabbitmq.list
wget -O- https://www.rabbitmq.com/rabbitmq-release-signing-key.asc | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y git make mercurial
sudo apt-get install -y rabbitmq-server
sudo rabbitmq-plugins enable rabbitmq_management

sudo wget -O /usr/local/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
sudo chmod +x /usr/local/bin/gimme
gimme 1.8 >> .bashrc

mkdir ~/go
eval "$(/usr/local/bin/gimme 1.8)"
echo 'export GOPATH=$HOME/go' >> .bashrc
export GOPATH=$HOME/go

export PATH=$PATH:$HOME/terraform:$HOME/go/bin
echo 'export PATH=$PATH:$HOME/terraform:$HOME/go/bin' >> .bashrc
source .bashrc

go get -u github.com/kardianos/govendor
go get github.com/hashicorp/terraform

cat <<EOF > ~/rabbitmqrc
export RABBITMQ_ENDPOINT="http://127.0.0.1:15672"
export RABBITMQ_USERNAME="guest"
export RABBITMQ_PASSWORD="guest"
EOF
