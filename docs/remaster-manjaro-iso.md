# Remastering a Manjaro ISO

From the [Manjaro wiki](https://wiki.manjaro.org/index.php?title=Manjaro-tools#buildiso)
it is unclear how the initramfs is compiled.

## Inspecting the ISO

The structure differs from the Arch ISO. There are multiple
squashfs files.

```sh
$ ls -l /mnt/iso/manjaro/x86_64/
total 2.6G
-r--r--r-- 1 root root   48 May 11 08:56 desktopfs.md5
-r--r--r-- 1 root root 1.3G May 11 08:56 desktopfs.sfs
-r--r--r-- 1 root root   45 May 11 09:40 livefs.md5
-r--r--r-- 1 root root  61M May 11 09:40 livefs.sfs
-r--r--r-- 1 root root   45 May 11 08:54 mhwdfs.md5
-r--r--r-- 1 root root 618M May 11 08:54 mhwdfs.sfs
-r--r--r-- 1 root root   45 May 11 08:58 rootfs.md5
-r--r--r-- 1 root root 585M May 11 08:58 rootfs.sfs
```

Extracting `rootfs.sfs` and following the approach for Arch errors:

```sh
$ mkinitcpio -k $(ls /lib/modules/) -p linux56
==> Building image from preset: /etc/mkinitcpio.d/linux56.preset: 'default'
  -> -k /boot/vmlinuz-5.6-x86_64 -c /etc/mkinitcpio.conf -g /boot/initramfs-5.6-x86_64.img
==> Starting build: 5.6.11-1-MANJARO
  -> Running build hook: [base]
  -> Running build hook: [udev]
  -> Running build hook: [memdisk]
==> ERROR: file not found: `memdiskfind'
==> ERROR: Hook 'archiso_shutdown' cannot be found
==> ERROR: Hook 'archiso' cannot be found
==> ERROR: Hook 'archiso_loop_mnt' cannot be found
==> ERROR: Hook 'archiso_pxe_common' cannot be found
==> ERROR: Hook 'archiso_pxe_nbd' cannot be found
==> ERROR: Hook 'archiso_pxe_http' cannot be found
==> ERROR: Hook 'archiso_pxe_nfs' cannot be found
==> ERROR: Hook 'archiso_kms' cannot be found
  -> Running build hook: [block]
  -> Running build hook: [filesystems]
  -> Running build hook: [keyboard]
  -> Running build hook: [nvdimm]
==> Generating module dependencies
==> Creating xz-compressed initcpio image: /boot/initramfs-5.6-x86_64.img
==> WARNING: errors were encountered during the build. The image may not be complete.
```

The hooks used for `archiso` are apparently not available.

The `mkinitcpio.conf` from `desktopfs.sfs` has `systemd` and `ostree`:

```
HOOKS="base systemd ostree autodetect modconf block filesystems keyboard fsck"
```

Which leads to no success either:

```sh
$ mkinitcpio -k 5.6.11-1-MANJARO -g /boot/initramfs.img
==> Starting build: 5.6.11-1-MANJARO
  -> Running build hook: [base]
  -> Running build hook: [systemd]
==> ERROR: Hook 'ostree' cannot be found
  -> Running build hook: [autodetect]
==> ERROR: failed to detect root filesystem
  -> Running build hook: [modconf]
  -> Running build hook: [block]
  -> Running build hook: [filesystems]
  -> Running build hook: [keyboard]
  -> Running build hook: [fsck]
  -> Running build hook: [nvdimm]
==> Generating module dependencies
==> Creating xz-compressed initcpio image: /boot/initramfs.img
==> WARNING: errors were encountered during the build. The image may not be complete.
```

## Booting the ISO for more insight

```sh
$ qemu-system-x86_64 -cdrom manjaro-gnome-20.0.1-200511-linux56.iso
```

### Kernel arguments

A look at `cat /proc/cmdline` yields the crucial bits to pass to the
kernel so that the initramfs can pick it up to find the ISO on devices.
The naming is different from Arch / SystemRescueCd: `misolabel` and
`misobasedir`.

```
misobasedir=manjaro misolabel=MANJARO_GNOME_2001
```

## Rebuilding the initramfs manually

The steps are simple, just require some knowledge:

- extract the initramfs
- copy the modules into it, `unxz` them
- recreate the initramfs

Luckily, the [Red Hat docs](https://access.redhat.com/solutions/24029)
explain how to recreate a `cpio` image, boiling down to the following:

```sh
$ find . | cpio --create --format='newc' > ../new.img
```

The compressed image needs a special alignment. We have that covered in
the [u-root README](https://github.com/u-root/u-root#compression):

```sh
$ xz --check=crc32 -9 --lzma2=dict=1MiB \
   --stdout /mnt/tmp/new.img \
   | sudo dd conv=sync bs=512 \
   of=initramfs-x86_64.img
```

Trying it out in QEMU though calls for trouble again:

```
$ qemu-system-x86_64 \
  -machine q35,accel=kvm -m 1G -append 'memmap=512M!512M' \ 
  -kernel manjaro-remastered/boot/vmlinuz-x86_64 \
  -initrd manjaro-remastered/boot/initramfs-x86_64.img
```

There is no `/dev/pmem*` device. Uh-oh! And the modules were not loaded
either - but why? An attempt to `modprobe nd_pmem` tells that the
module cannot be found. But the file is present. A quick research says
that the module lookup needs some help. The `depmod` utility will
receate the `modules.dep` file in `/usr/lib/$KERNEL/`.
Solution: simply add `depmod` to the top of `/init`.

Recreating the initramfs now leads to an environment with the desired
pmem device present.

## Recreating the ISO

First, copy the new initramfs to a copy is the ISO in some directory.

Long story short: It's not `syslinux` like Arch does it, life is short.
The resulting command is:

```sh
$ genisoimage \
  -l -r -J -V "MANJARO_WEBBOOT" \
  -b efi/boot/bootx64.efi \
  -no-emul-boot -boot-load-size 4 -boot-info-table \
  -c boot.catalog \
  -o ../manjaro.iso \
  ./
```

This loses the ability to boot via QEMU directly, which is a shortcut
here for the proof of concept. For a full solution, the modules would
just be added to the initramfs by the upstream distribution anyway.

Now `webboot` successfully boots Manjaro with a full Gnome dekstop. :)

## TODOs

After some more search, here are the hooks used to create the Manjaro initramfs:
https://gitlab.manjaro.org/tools/development-tools/manjaro-tools/-/tree/master/initcpio/hooks

File a PR there to include another hook for adding the nvdimm/pmem modules.
