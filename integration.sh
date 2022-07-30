#!/bin/bash
#
# Intended for CI but can be run standalone.
#
# Runs an integration test using the distro name passed as the first argument.
# The list of accepted distros is in "integration/basic_test.go".
#
# Example usage:
#   cd /path/to/webboot/repo
#   sudo apt-get update
#   ./firsttime.sh
#   ./integration.sh TinyCore

# Exit immediately if a command fails
set -e

if [[ $# -eq 0 ]] ; then
    echo "integration.sh: No argument supplied. Please specify a distro as the first argument." 1>&2
    exit 1
fi

(
  cd integration

  # Download if bzImage not present
  wget --no-clobber https://github.com/u-root/webboot-distro/raw/master/CIkernels/5.6.14/bzImage

  # Print full path
  ls -dl "$PWD"/bzImage

  # Tests distro specified by first argument
  WEBBOOT_DISTRO="$1" \
  UROOT_QEMU="qemu-system-x86_64" \
  UROOT_KERNEL="$PWD/bzImage" \
  UROOT_INITRAMFS="/tmp/initramfs.linux_amd64.cpio" \
  go test -v
)
