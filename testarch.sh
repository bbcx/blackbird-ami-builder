#!/bin/bash

# harvest software versions
pacman -Q linux-ec2 2> /dev/null
pacman -Q systemd 2> /dev/null
pacman -Q kubernetes 2> /dev/null
pacman -Q etcd 2> /dev/null
pacman -Q rkt 2> /dev/null
pacman -Q docker 2> /dev/null

systemctl status

echo testing complete.
