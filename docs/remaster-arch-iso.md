# Remaster Arch Linux Live ISO for webboot

## Preparation

In general, follow the steps as described in
[the wiki](https://wiki.archlinux.org/index.php/Remastering_the_Install_ISO).

For webboot, the following steps are necessary:

- mount ISO
- extract squashfs
- copy over kernel
- copy over [mkinitcpio.conf](https://git.archlinux.org/archiso.git/tree/configs/releng/mkinitcpio.conf) (overwrite `squashfs-root/etc/mkinitcpio.conf`)
- chroot into rootfs
- [build new initramfs](#building-the-new-initramfs-in-the-chroot)
- copy mounted ISO
- add the new initramfs to it (`iso/arch/boot/x86_64/archiso.img`)
- build new ISO

## Building the new initramfs in the chroot

Arch has a tool to build a suitable initramfs that is used to create the ISO
images in the first place. So instead of extracting the existing initramfs,
hacking on it, and rebuilding it, `mkinitcpio` will do it. There are two
possible strategies:

1. add the pmem modules to the list of files to include (more generic)
2. write a custom hook to do it (https://github.com/archlinux/mkinitcpio/pull/30)

### Manual strategy

#### Decompress the modules

This is necessary because the initramfs cannot load compressed modules.

```sh
cd /lib/modules/5.6.8-arch1-1/kernel/drivers/nvdimm/
unxz nd_btt.ko.xz
unxz nd_e820.ko.xz
unxz nd_pmem.ko.xz
```

#### Adjust the mkinitcpio config

Add `FILES` to `/etc/mkinitcpio.conf`:

```
HOOKS=(base udev memdisk archiso_shutdown archiso archiso_loop_mnt archiso_pxe_common archiso_pxe_nbd archiso_pxe_http archiso_pxe_nfs archiso_kms block filesystems keyboard)
COMPRESSION="xz"
### the above is from the copied releng/mkinitcpio.conf
FILES=(
/lib/modules/5.6.8-arch1-1/kernel/drivers/nvdimm/nd_btt.ko
/lib/modules/5.6.8-arch1-1/kernel/drivers/nvdimm/nd_e820.ko
/lib/modules/5.6.8-arch1-1/kernel/drivers/nvdimm/nd_pmem.ko)
```

### Using a custom hook

Create the file `/lib/initcpio/install/nvdimm`:

```sh
#!/bin/bash

build() {
    add_checked_modules '/drivers/nvdimm/'
}
```

And add it to the `HOOKS` in `/etc/mkinitcpio.conf`:

```
HOOKS=(base udev memdisk archiso_shutdown archiso archiso_loop_mnt archiso_pxe_common archiso_pxe_nbd archiso_pxe_http archiso_pxe_nfs archiso_kms block filesystems keyboard nvdimm)
COMPRESSION="xz"
```

### Create the new initramfs

```sh
_preset=`ls /etc/mkinitcpio.d/|sed 's#\..*##'`
mkinitcpio -k $(ls /lib/modules/) -p $preset`
```

## Try it out

Check that everything works before rebuilding the ISO and trying to boot it.

The location where you extracted the squashfs root may differ. This assumes
following the guide, using the `run-webboot.sh` script from the repo:

```sh
$ sh run-webboot.sh \
  ~/customiso/arch/x86_64/squashfs-root/boot/vmlinuz-linux \
  ~/customiso/arch/x86_64/squashfs-root/boot/initramfs.img
```

Check that the modules are present:

```
[rootfs ]# ls /lib/modules/5.6.8-arch1-1/kernel/drivers/nvdimm/
nd_btt.ko   nd_e820.ko  nd_pmem.ko
```

The modules should be loaded because `memmap` was passed from our script:

```
[rootfs ]# lsmod
Module                  Size  Used by
nd_pmem                24576  0
nd_btt                 28672  1 nd_pmem
serio_raw              20480  0
atkbd                  36864  0
libps2                 20480  1 atkbd
nd_e820                16384  1
sr_mod                 28672  0
cdrom                  77824  1 sr_mod
i8042                  32768  0
serio                  28672  5 serio_raw,atkbd,i8042
```

And you should get a pmem device:

```
[rootfs ]# dmesg|grep pmem
[    9.013750] nd_pmem namespace0.0: unable to guarantee persistence of writes
[    9.172887] pmem0: detected capacity change from 0 to 805306368
```

If you get a confusing error in dmesg, try smaller memmap sizes.
Some people wrote that steps of 64M or 256M should work.
I had success with 768M, barely enough for the Arch ISO, and 4G, enough for
bigger live environments including full graphical UIs such as Gnome or KDE.

See also https://github.com/pmem/ndctl/issues/76#issuecomment-440849415.

## Rebuild the ISO

Generally, follow the wiki. Mind the importance of the label of the ISO.
We need to pass it when we kexec. The Arch ISO itself assumes booting
through its own bootloader, and the hooks in its initramfs pick it up.
In other words: Whatever label you choose, use it also in `webboot.go`.

See the file `arch/boot/syslinux/archiso_sys.cfg` from the Arch ISO to see what
the initramfs expects from the bootloader.
