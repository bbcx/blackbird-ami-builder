#!/bin/bash

# harvest software versions
# output markdown table to stdout
linux_version=`pacman -Q linux-ec2 2> /dev/null| cut -f2 -d" "`
kube_version=`pacman -Q kubernetes 2> /dev/null| cut -f2 -d" "`
systemd_version=`pacman -Q systemd 2> /dev/null| cut -f2 -d" "`
etcd_version=`pacman -Q etcd 2> /dev/null| cut -f2 -d" "`
rkt_version=`pacman -Q rkt 2> /dev/null| cut -f2 -d" "`
docker_version=`pacman -Q docker 2> /dev/null| cut -f2 -d" "`
go_version=`pacman -Q go 2> /dev/null |cut -f2 -d":"`

echo "| Package | Version |"
echo "|---:|---:|"
echo "| linux-ec2 | \`${linux_version}\` |"
echo "| go | \`${go_version}\` |"
echo "| systemd | \`${systemd_version}\` |"
echo "| kubernetes | \`${kube_version}\` |"
echo "| etcd | \`${etcd_version}\` |"
echo "| docker | \`${docker_version}\` |"
echo "| rkt | \`${rkt_version}\` |"
