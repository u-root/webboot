#!/bin/bash
set -e

# Check that the code has been formatted correctly.
GOFMT_DIFF=$(gofmt -s -d *.go pkg cmds)
if [[ -n "${GOFMT_DIFF}" ]]; then
	echo 'Error: Go source code is not formatted:'
	printf '%s\n' "${GOFMT_DIFF}"
	echo 'Run `gofmt -s -w *.go pkg cmds'
	exit 1
fi

go mod tidy
go build .
./webboot
if [ ! -f "/tmp/initramfs.linux_amd64.cpio" ]; then
    echo "Initrd was not created."
    exit 1
fi

(cd cmds/webboot && go test -v)
(cd pkg/menu && go test -v)
(cd pkg/bootiso && sudo -E env "PATH=$PATH" go test -v) # need sudo to mount the test iso
