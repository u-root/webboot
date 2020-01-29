#!/bin/sh

# https://docs.pmem.io/persistent-memory/getting-started-guide/creating-development-environments/virtualization/qemu
# PMEM *must* be less than RAM so the system has some memory to work with
export PMEM_SIZE=1G
export RAM_SIZE=4G
export KERNEL=$1
export INITRD=$2

qemu-system-x86_64 \
  -machine q35 \
  -m $RAM_SIZE \
  -object rng-random,filename=/dev/urandom,id=rng0 \
  -device virtio-rng-pci,rng=rng0 \
  -netdev user,id=network0 -device rtl8139,netdev=network0 \
  -kernel $KERNEL \
  -initrd $INITRD \
  -append "console=ttyS0 vga=786 memmap=$PMEM_SIZE!$PMEM_SIZE" \
  -serial stdio
