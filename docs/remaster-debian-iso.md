# Remastering Debian live ISO

## Figure out kernel command line
`less /mnt/iso/boot/grub/grub.cfg`

```
...
  linux  /live/vmlinuz-4.19.0-9-amd64 boot=live components splash quiet "${loopback}"
  initrd /live/initrd.img-4.19.0-9-amd64
...
```

## Tailor initrd for webboot

### Extract initrd
```sh
zcat /mnt/iso/live/initrd.img-4.19.0-9-amd64 | sudo cpio -idmv
```

### Extract and copy nvdimm modules from squashfs
```sh
unsquashfs -f /mnt/iso/live/filesystem.squashfs \
  -e usr/lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm
mv squashfs-root/usr/lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm \
  lib/modules/4.19.0-9-amd64/kernel/drivers
rm -r squashfs-root
```

### Add insmod statements to /init - little hackaround :)
```sh
insmod /lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm/libnvdimm.ko
insmod /lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm/nd_btt.ko
insmod /lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm/nd_pmem.ko
insmod /lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm/nd_e820.ko
insmod /lib/modules/4.19.0-9-amd64/kernel/drivers/nvdimm/nd_blk.ko
```

### Rebuild initrd
```sh
find . | cpio --create --format='newc' | gzip > initrd
```

## Build new ISO

```sh
cp -a /mnt/iso/ debian-remastered/
cp initrd.img-4.19.0-9-amd64 debian-remastered/live/initrd.img-4.19.0-9-amd64
cd debian-remastered
genisoimage \
  -l -r -J -V "DEBIAN_WEBBOOT" \
  -b isolinux/isolinux.bin \
  -no-emul-boot -boot-load-size 4 -boot-info-table \
  -c isolinux/boot.cat \
  -o ../debian.iso \
  ./
```
