#!/bin/bash
set -e

go build .
go run webboot.go
if [ ! -f "/tmp/initramfs.linux_amd64.cpio" ]; then
    echo "Initrd was not created."
    exit 1
fi

(cd cmds/webboot && go test -v)
(cd pkg/menu && go test -v)
(cd pkg/bootiso && sudo -E env "PATH=$PATH" go test -v) # need sudo to mount the test iso
