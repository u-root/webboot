#!/bin/bash
git clone --depth 1 -b v5.6.14 git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git linux
git clone git://git.kernel.org/pub/scm/linux/kernel/git/iwlwifi/linux-firmware.git
cp config-5.6.14 linux/.config
(cd linux && make bzImage)
