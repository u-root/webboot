#!/bin/sh
set -e
go run . 
mkdir -p /mnt/usb
sudo mount /dev/$1 /mnt/usb
gzip -f /tmp/initramfs.linux_amd64.cpio 
sudo cp /tmp/initramfs.linux_amd64.cpio.gz /mnt/usb/boot/webboot.cpio.gz
sudo umount /mnt/usb

