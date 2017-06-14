#!/bin/bash

set -o errexit -o nounset

if docker -v; then

  # generate a unique string for CI deployment
  export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
  export PASSWORD="P4ssw0rd1"
  export KEY_VAULT_RESOURCE_GROUP=permanent
  export KEY_VAULT_NAME=TerraformVault
  export KEY_VAULT_SECRET=OpenShiftSSH
  export OS_PUBLIC_KEY='ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCvdOGspeeBhsOZw6OK2WmP3bBUOeZj1yaz6Dw+lzsRmjwNSmJIoGZPzlbdy1lzlkXIm2JaT4h/cUi39w+Q2RZRjxmr7TbLyuidJfFLvRJ35RDullUYLWEPx3csBroPkCv+0qgmTW/MqqjqS4yhlJ01uc9RNx9Jt3XZN7LNr8SUoBzdLCWJa1rpCTtUckO1Jyzi4VwZ2ek+nYPJuJ8hG0KeHnyXDXV4hQZTFtGvtbmgoyoybppFQMbM3a31KZeaWXUeZkZczBsdNRkX8XCDjb6zUmUMQUzZpalFlL1O+rZD0kaXKr0uZWiYOKu2LjnWeDW9x4tig1mf+L84vniP+lLKFW8na3Lzx11ysEpuhIJGPMMI8sjTCnu51PmiwHW2U9OR06skPUO7ZGD0QHg7jKXdz5bHT+1OqXeAStULDiPVRIPrxxpurPXiJRm7JPbPvPqrMqZJ3K7J9W6OGHG3CoDR5RfYlPWURTaVH10stb4hKevasCd+YoLStB1XgMaL/cG9bM0TIWmODV/+pfn800PgxeBn1vABpL0NF8K2POLs37vGJoh/RyGCDVd0HEKArpZj0/g+fv7tr3tFFOCY5bHSuDTZcY8sWPhxKXSismoApM3a+USF5HkDkWSTEiETs2wgUdTSt4MuN2maRXOK2JboQth1Qw+vCOvqcls0dMa0NQ== you@example.com'
  export CONTAINER_PRIVATE_KEY_PATH="/data/Users/$USER/.ssh/id_rsa"
  export LOCAL_SCRIPT_PATH="/data/Users/$USER/Code/10thmagnitude/openshift-origin/scripts"
  export MASTER_COUNT=1
  export INFRA_COUNT=1
  export NODE_COUNT=1

/bin/sh ./deploy.ci.sh

else
  echo "Docker is used to run terraform commands, please install before run:  https://docs.docker.com/docker-for-mac/install/"
fi