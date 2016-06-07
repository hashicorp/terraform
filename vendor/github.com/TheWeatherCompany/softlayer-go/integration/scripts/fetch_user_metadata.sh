#!/bin/sh

mkdir -p /tmp/config

mount /dev/xvdh1 /tmp/config/

DECODED_USERDATA=`cat /tmp/config/openstack/latest/user_data | base64 -d`

echo $DECODED_USERDATA

if [ "$DECODED_USERDATA" = "$1" ] 
then
	exit 0
else
	exit 1
fi