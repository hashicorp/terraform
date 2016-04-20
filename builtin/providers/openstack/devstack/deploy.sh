#!/bin/bash

set -e

cd
sudo apt-get update
sudo apt-get install -y git make mercurial

sudo wget -O /usr/local/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
sudo chmod +x /usr/local/bin/gimme
gimme 1.6 >> .bashrc

mkdir ~/go
eval "$(/usr/local/bin/gimme 1.6)"
echo 'export GOPATH=$HOME/go' >> .bashrc
export GOPATH=$HOME/go

export PATH=$PATH:$HOME/terraform:$HOME/go/bin
echo 'export PATH=$PATH:$HOME/terraform:$HOME/go/bin' >> .bashrc
source .bashrc

go get github.com/tools/godep
go get github.com/hashicorp/terraform
cd $GOPATH/src/github.com/hashicorp/terraform
godep restore

cd
git clone https://git.openstack.org/openstack-dev/devstack -b stable/mitaka
cd devstack
cat >local.conf <<EOF
[[local|localrc]]
# OpenStack version
OPENSTACK_VERSION="mitaka"

# devstack password
DEVSTACK_PASSWORD="password"

# Configure passwords and the Swift Hash
MYSQL_PASSWORD=\$DEVSTACK_PASSWORD
RABBIT_PASSWORD=\$DEVSTACK_PASSWORD
SERVICE_TOKEN=\$DEVSTACK_PASSWORD
ADMIN_PASSWORD=\$DEVSTACK_PASSWORD
SERVICE_PASSWORD=\$DEVSTACK_PASSWORD
SWIFT_HASH=\$DEVSTACK_PASSWORD

# Configure the stable OpenStack branches used by DevStack
# For stable branches see
# http://git.openstack.org/cgit/openstack-dev/devstack/refs/
CINDER_BRANCH=stable/\$OPENSTACK_VERSION
CEILOMETER_BRANCH=stable/\$OPENSTACK_VERSION
GLANCE_BRANCH=stable/\$OPENSTACK_VERSION
HEAT_BRANCH=stable/\$OPENSTACK_VERSION
HORIZON_BRANCH=stable/\$OPENSTACK_VERSION
KEYSTONE_BRANCH=stable/\$OPENSTACK_VERSION
NEUTRON_BRANCH=stable/\$OPENSTACK_VERSION
NOVA_BRANCH=stable/\$OPENSTACK_VERSION
SWIFT_BRANCH=stable/\$OPENSTACK_VERSION
ZAQAR_BRANCH=stable/\$OPENSTACK_VERSION

# Enable Swift
enable_service s-proxy
enable_service s-object
enable_service s-container
enable_service s-account

# Disable Nova Network and enable Neutron
disable_service n-net
enable_service q-svc
enable_service q-agt
enable_service q-dhcp
enable_service q-l3
enable_service q-meta
enable_service q-metering
enable_service q-lbaas
enable_service q-fwaas

# Enable Trove
enable_plugin trove git://git.openstack.org/openstack/trove.git stable/\$OPENSTACK_VERSION
enable_service trove,tr-api,tr-tmgr,tr-cond

# Disable Temptest
disable_service tempest

# Disable Horizon
disable_service horizon

# Enable Ceilometer
#enable_service ceilometer-acompute
#enable_service ceilometer-acentral
#enable_service ceilometer-anotification
#enable_service ceilometer-collector
#enable_service ceilometer-alarm-evaluator
#enable_service ceilometer-alarm-notifier
#enable_service ceilometer-api

# Enable Zaqar
#enable_plugin zaqar https://github.com/openstack/zaqar
#enable_service zaqar-server

# Automatically download and register a VM image that Heat can launch
# For more information on Heat and DevStack see
# http://docs.openstack.org/developer/heat/getting_started/on_devstack.html
#IMAGE_URLS+=",http://cloud.fedoraproject.org/fedora-20.x86_64.qcow2"
#IMAGE_URLS+=",https://cloud-images.ubuntu.com/trusty/current/trusty-server-cloudimg-amd64-disk1.img"

# Logging
LOGDAYS=1
LOGFILE=/opt/stack/logs/stack.sh.log
LOGDIR=/opt/stack/logs
EOF
./stack.sh

# Prep the testing environment by creating the required testing resources and environment variables
source openrc admin
wget http://download.cirros-cloud.net/0.3.4/cirros-0.3.4-x86_64-disk.img
glance image-create --name CirrOS --disk-format qcow2 --container-format bare < cirros-0.3.4-x86_64-disk.img
nova flavor-create m1.tform 99 512 5 1 --ephemeral 10
_NETWORK_ID=$(nova net-list | grep private | awk -F\| '{print $2}' | tr -d ' ')
_IMAGE_ID=$(nova image-list | grep CirrOS | awk -F\| '{print $2}' | tr -d ' ' | head -1)
echo export OS_IMAGE_NAME="cirros-0.3.4-x86_64-uec" >> openrc
echo export OS_IMAGE_ID="$_IMAGE_ID" >> openrc
echo export OS_NETWORK_ID=$_NETWORK_ID >> openrc
echo export OS_POOL_NAME="public" >> openrc
echo export OS_FLAVOR_ID=99 >> openrc
source openrc demo

# Replace the below lines with the repo/branch you want to test
#git remote add jtopjian https://github.com/jtopjian/terraform
#git fetch jtopjian
#git checkout --track jtopjian/openstack-secgroup-safe-delete
#make testacc TEST=./builtin/providers/openstack TESTARGS='-run=AccBlockStorageV1'
#make testacc TEST=./builtin/providers/openstack TESTARGS='-run=AccCompute'
#make testacc TEST=./builtin/providers/openstack
