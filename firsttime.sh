#!/bin/bash
set -e

sudo apt-get install wireless-tools wpasupplicant libnl-3-dev libnl-genl-3-dev

go get -u github.com/u-root/u-root
go get -u github.com/u-root/NiChrome/...
