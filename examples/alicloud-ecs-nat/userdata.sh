#!/bin/sh

PostRouting=${vswitch_cidr}
SourceRouting=`ifconfig eth0|grep inet|awk '{print $2}'|tr -d 'addr:'`
echo ${worker_private_ip}>> /etc/sysctl.conf
echo 'net.ipv4.ip_forward=1'>> /etc/sysctl.conf
sysctl -p
iptables -t nat -I POSTROUTING -s $PostRouting -j SNAT --to-source $SourceRouting
iptables -t nat -I PREROUTING -d $SourceRouting -p tcp --dport 80 -j DNAT --to ${worker_private_ip}