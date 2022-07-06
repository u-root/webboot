#!/bin/bash
set -e

sudo apt-get install build-essential kexec-tools libelf-dev libnl-3-dev libnl-genl-3-dev libssl-dev qemu-system-x86 wireless-tools wpasupplicant

pwd
ls
git clone https://github.com/u-root/u-root.git ../u-root
(cd ../u-root/ && go install .)
