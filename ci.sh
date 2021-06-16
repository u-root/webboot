#!/bin/bash
set -e

# Check that the code has been formatted correctly.
test -z "$(gofmt -s -l *.go pkg cmds)"

go build .
go run webboot.go --wpa-version=2.9
if [ ! -f "/tmp/initramfs.linux_amd64.cpio" ]; then
    echo "Initrd was not created."
    exit 1
fi

(cd cmds/webboot && go test -v)
(cd pkg/menu && go test -v)
(cd pkg/bootiso && sudo -E env "PATH=$PATH" go test -v) # need sudo to mount the test iso
