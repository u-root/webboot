#!/bin/bash
set -e

sudo apt-get install build-essential kexec-tools libelf-dev libnl-3-dev libnl-genl-3-dev libssl-dev qemu-system-x86 wireless-tools wpasupplicant

go get -u github.com/u-root/u-root
go get -u github.com/u-root/NiChrome/...
