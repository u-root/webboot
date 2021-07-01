#!/bin/bash
set -e

sudo apt-get install wireless-tools
sudo apt-get install wpasupplicant

go get -u github.com/u-root/u-root
go get -u github.com/u-root/NiChrome/...
