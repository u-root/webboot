# About

`webboot` offers tools to let a u-root instance boot signed live distro images over the web.

## Concept

The `webboot` bootloader works as follows:

1. fetch an OS distro release ISO from the web
2. save the ISO to a local cache (ex. USB stick)
3. mount the ISO and copy out the kernel and initrd
4. load the extracted kernel with the initrd
5. kexec that kernel with parameters to tell the next distro where to locate its ISO file (ex. iso-scan/filename=)

The current version offers a user interface based on [termui](https://github.com/gizak/termui) to help locate and boot the ISO file.

For reference, webboot developers should familiarize themselves with:

- [cpio tutorial](https://www.gnu.org/software/cpio/manual/html_node/Tutorial.html)
- [initrd usage](https://www.kernel.org/doc/html/latest/admin-guide/initrd.html)
- [kernel parameters](https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html)


## Supported Operating Systems

### Requirements
ISOs must have the following to be fully compatible with `webboot`.

1. 64-bit kernel
2. Parsable `grub` or `syslinux` config file
3. Init process than can locate an ISO file (ex. casper's iso-scan)

Additional operating systems can be added by appending an entry to the `supportedDistros` map in `/cmds/webboot/types.go`.

If the config file is not compatible with our parser, we can manually specify the configuration by adding a `Config` object to the distro's entry in `supportedDistros`. See the entries for Arch and Manjaro as an example.

### Currently Supported
| Name | Required Kernel Parameters | Notes |
| ----- | ------ | ----- |
| Arch | `img_dev=/dev/disk/by-uuid/UUID img_loop=PATH_TO_ISO` | Unable to parse config file. Configuration is specified in a `Config` object.
| CentOS | `iso-scan/filename=PATH_TO_ISO` | CentOS 7 supports live mode. CentOS 8 will boot to the graphical installer.
| Debian | `findiso=PATH_TO_ISO` |
| Fedora | `iso-scan/filename=PATH_TO_ISO` |
| Kali | `findiso=PATH_TO_ISO` |
| Linux Mint | `iso-scan/filename=PATH_TO_ISO` |
| Manjaro | `img_dev=/dev/disk/by-uuid/UUID img_loop=PATH_TO_ISO` | Unable to parse config file. Configuration is specified in a `Config` object.
| Tinycore | `iso=UUID/PATH_TO_ISO` |
| Ubuntu | `iso-scan/filename=PATH_TO_ISO` |

### In Progress
| Name | Required Kernel Parameters | Issue |
| --- | --- | --- |
| OpenSUSE | `root=live:CDLABEL=ISO_LABEL iso-scan/filename=PATH_TO_ISO` | `grub` config file is too complicated for our parser. We could specify the configuration manually, but that would involve hardcoding the ISO_LABEL (see [Issue 185](https://github.com/u-root/webboot/issues/185)).|

## Usage

### Build initramfs with added webboot commands

Download u-root with `GO111MODULE=off go get github.com/u-root/u-root`.

Run `GO111MODULE=off go run .` in the source directory of webboot to build the
initramfs.

This runs [u-root](https://github.com/u-root/u-root) under the hood. To pass
extra options, such as to include extra files, use the `-u` switch, e.g.,
`GO111MODULE=off go run buildimage.go -u "-files path/to/bzImage:bzImage"` to
add a custom kernel which can be used to test whether kexec works in a small
setup. That saves a lot of time, because a full webboot flow would always need
to download large ISO files, copy them, mount and decompress.

#### Convenience

For convenience, you can

- skip the inclusion of Wi-Fi tools by passing `-wifi false`
- add a custom kernel for within the initramfs via `-bzImage path/to/bzImage`
- add an ISO file to the initramfs via `-iso path/to/os-distro.iso`
    * boot that ISO via `webboot -dhcp4=false -dhcp6-false local` later, which
      requires passing a pmem-enabled kernel via `-bzImage` as described above

#### Compression

You can optionally compress the initramfs with `lzma` or any other compression
method you configure your kernel for.

```sh
lzma -f /tmp/initramfs.linux_amd64.cpio
```

Refer to [u-root's documentation](https://github.com/u-root/u-root#compression)
for more details on compression.

#### Customization

The `buildimage.go` utility is really just a helper tool. Instead of using it,
you can build a custom u-root image as you like and add the `webboot` binary to
it.
Refer to [u-root's usage documentation](https://github.com/u-root/u-root#usage)
for details.

### Building a kernel for webboot

webboot uses a standard Linux kernel which should be fairly portable, based on a
Long Term Stable (LTS) release. It has worked on every Chromebook we tried.

This kernel is built using a config originally from
[NiChromeOS](github.com/NiChrome/NiChrome).
If we are building a bootable USB stick formatted with vfat, we don't have the
space constraints of NiChrome, so we expect this to diverge over time.

Nevertheless, to keep it all simple, we build it as a non-modular kernel with
Wi-fi firmware built-in. We no longer build the initramfs into the kernel, as
that's not needed.

Make sure the kernel configuration includes the firmware for your network device.
For instance, the Thinkpad x240 with Intel Corporation Wireless 7260 uses
iwlwifi-7260-17.ucode. If you look at the kernel config file, this firmware name
is included under `CONFIG_EXTRA_FIRMWARE=`.

To build, first be sure you're in a directory you want to be in!
You can actually do the work in the webboot root directory because the
`.gitignore` file ignores the two directories you create when following the
instructions here.

#### Prerequisites

You need to have the following packages installed if on Ubuntu:
```sh
sudo apt install libssl-dev build-essential wireless-tools kexec-tools libelf-dev libnl-3-dev libnl-genl-3-dev
```

#### Fetching, configuring and compiling the kernel

```sh
git clone --depth 1 -b v5.6.14 \
  git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git linux
git clone \
  git://git.kernel.org/pub/scm/linux/kernel/git/iwlwifi/linux-firmware.git
cp config-5.6.14 linux/.config
(cd linux && make bzImage)
GO111MODULE=off go run .
```

### Testing in QEMU

Install QEMU with
```sh
sudo apt-get install qemu-system-x86
```

Run the following, and a QEMU window should pop up:

Tip: Don't use the `-nographic` option for u-root in QEMU as you want to boot
into a graphical interface.

#### Acceleration

If you have KVM in your host system, you can add `-enable-kvm` for speedup.

```sh
qemu-system-x86_64 \
  -enable-kvm \
  -m 2G \
  -kernel linux/arch/x86/boot/bzImage \
  -append 'console=ttyS0 console=tty1 memmap=1G!1G' \
  -initrd /tmp/initramfs.linux_amd64.cpio \
  -device virtio-rng-pci \
  -netdev user,id=network0 \
  -device rtl8139,netdev=network0
```

Tip: Don't use the `-nographic` option for u-root in QEMU as you want to boot
into a graphical interface.

In the QEMU terminal, run
```sh
dhclient -ipv6=f
```
and then run
```sh
webboot
```

Refer to
[u-root's documentation](https://github.com/u-root/u-root#testing-in-qemu) for
more details on virtualization.

### Testing with a USB stick

You can try out webboot from a USB stick. That means that you could run it when
starting a machine by choosing to boot from USB, which requires a bootloader.
Although any bootloader would do, we will focus on one here named `syslinux`.
Furthermore, we will focus on specific preconditions, although there are many
different ways to create a bootable USB stick.

In the root directory of this repository, there is an example configuration file
named `syslinux.cfg.example`. If you look at it, you will see that it resembles
webboot very much: It lists a kernel, an initrd, and extra arguments to append.

Before you continue, please make sure to meet the following conditions:

- your system can boot from MBR (possibly through UEFI CSM)
- You have a directory `/mnt/usb` to mount the partition to

To [install](https://wiki.syslinux.org/wiki/index.php?title=Install) syslinux as
a bootloader and configure it, four steps are necessary:

1. Write a Volume Boot Record (VBR) to the stick
2. Write a Master Boot Record (MBR) to it
3. Mark the first partition as bootable
4. Copy the config file, Linux kernel, and initcpio

The following instructions will walk you through these four steps.
Tip: You may need to replace `sdb1` with the name of your partition.

Install syslinux with
```sh
sudo apt-get install syslinux
```

To prepare your USB stick, run `sudo fdisk /dev/sdb` and use the fdisk instructions to complete the following:
1. Delete all existing partitions (d)
2. Add one new partition (n, p, 1)
3. Change partition type (t) to EFI (ef)
4. Make partition 1 bootable (a)
5. Save (w)

Here is a sample fdisk output:
```sh
$ sudo fdisk /dev/sdb

Welcome to fdisk (util-linux 2.36.1).
Changes will remain in memory only, until you decide to write them.
Be careful before using the write command.


Command (m for help): d
Selected partition 1
Partition 1 has been deleted.

Command (m for help): n
Partition type
   p   primary (0 primary, 0 extended, 4 free)
   e   extended (container for logical partitions)
Select (default p): p
Partition number (1-4, default 1): 1
First sector (2048-121061375, default 2048):
Last sector, +/-sectors or +/-size{K,M,G,T,P} (2048-121061375, default 121061375):

Created a new partition 1 of type 'Linux' and of size 57.7 GiB.

Command (m for help): t
Selected partition 1
Hex code or alias (type L to list all): L

00 Empty            24 NEC DOS          81 Minix / old Lin  bf Solaris
01 FAT12            27 Hidden NTFS Win  82 Linux swap / So  c1 DRDOS/sec (FAT-
02 XENIX root       39 Plan 9           83 Linux            c4 DRDOS/sec (FAT-
03 XENIX usr        3c PartitionMagic   84 OS/2 hidden or   c6 DRDOS/sec (FAT-
04 FAT16 <32M       40 Venix 80286      85 Linux extended   c7 Syrinx
05 Extended         41 PPC PReP Boot    86 NTFS volume set  da Non-FS data
06 FAT16            42 SFS              87 NTFS volume set  db CP/M / CTOS / .
07 HPFS/NTFS/exFAT  4d QNX4.x           88 Linux plaintext  de Dell Utility
08 AIX              4e QNX4.x 2nd part  8e Linux LVM        df BootIt
09 AIX bootable     4f QNX4.x 3rd part  93 Amoeba           e1 DOS access
0a OS/2 Boot Manag  50 OnTrack DM       94 Amoeba BBT       e3 DOS R/O
0b W95 FAT32        51 OnTrack DM6 Aux  9f BSD/OS           e4 SpeedStor
0c W95 FAT32 (LBA)  52 CP/M             a0 IBM Thinkpad hi  ea Linux extended
0e W95 FAT16 (LBA)  53 OnTrack DM6 Aux  a5 FreeBSD          eb BeOS fs
0f W95 Ext'd (LBA)  54 OnTrackDM6       a6 OpenBSD          ee GPT
10 OPUS             55 EZ-Drive         a7 NeXTSTEP         ef EFI (FAT-12/16/
11 Hidden FAT12     56 Golden Bow       a8 Darwin UFS       f0 Linux/PA-RISC b
12 Compaq diagnost  5c Priam Edisk      a9 NetBSD           f1 SpeedStor
14 Hidden FAT16 <3  61 SpeedStor        ab Darwin boot      f4 SpeedStor
16 Hidden FAT16     63 GNU HURD or Sys  af HFS / HFS+       f2 DOS secondary
17 Hidden HPFS/NTF  64 Novell Netware   b7 BSDI fs          fb VMware VMFS
18 AST SmartSleep   65 Novell Netware   b8 BSDI swap        fc VMware VMKCORE
1b Hidden W95 FAT3  70 DiskSecure Mult  bb Boot Wizard hid  fd Linux raid auto
1c Hidden W95 FAT3  75 PC/IX            bc Acronis FAT32 L  fe LANstep
1e Hidden W95 FAT1  80 Old Minix        be Solaris boot     ff BBT

Aliases:
   linux          - 83
   swap           - 82
   extended       - 05
   uefi           - EF
   raid           - FD
   lvm            - 8E
   linuxex        - 85
Hex code or alias (type L to list all): EF
Changed type of partition 'Linux' to 'EFI (FAT-12/16/32)'.

Command (m for help): a
Selected partition 1
The bootable flag on partition 1 is enabled now.

Command (m for help): w
The partition table has been altered.
Calling ioctl() to re-read partition table.
Syncing disks.
```

Generate the partition header
```sh
mkfs -t vfat /dev/sdb1
```

Mount the USB and copy the config file, Linux kernel, and initcpio
```sh
sudo mount /dev/sdb1 /mnt/usb
cp config-5.6.14 /mnt/usb/
cp arch/x86/boot/bzImage /mnt/usb
cp /tmp/initramfs.linux_amd64.cpio /mnt/usb
umount /mnt/usb
```

Zip initramfs
```sh
gzip /tmp/initramfs.linux_amd64.cpio
```

Now the following commands would need to be run as root:

```sh
syslinux -i /dev/sdb1
dd bs=440 count=1 conv=notrunc if=/usr/lib/syslinux/mbr/mbr.bin of=/dev/sdb
parted /dev/sdb set 1 boot on
# mount the stick and copy the files
mount /dev/sdb1 /mnt/usb
cp syslinux.cfg.example /mnt/usb/syslinux.cfg
mkdir /mnt/usb/boot
cp linux/arch/x86/boot/bzImage /mnt/usb/boot/webboot
cp /tmp/initramfs.linux_amd64.cpio.gz /mnt/usb/boot/webboot.cpio.gz
```

Finally, we need to create a `/Images` directory at the root of the usb stick. Note that the "I" in "Images" needs to be capitalized.

```sh
mkdir /mnt/usb/Images
```

You should be able to boot from the USB stick now. Depending on your firmware
setup, it might be necessary to get into a boot menu or make changes in the
settings.

## Legacy Information

**Note: The following information does not apply as of [PR 156](https://github.com/u-root/webboot/pull/156). However the original concept for webboot, as laid out below, is worth noting.**

### Original Concept (using pmem)

The original concept for webboot was the following:
1. fetch an OS distro release ISO from the web
2. copy the ISO to memory
3. mount the ISO and copy out the kernel and initrd
4. load the extracted kernel with the initrd
5. kexec that kernel with [memmap parameters](https://docs.pmem.io/persistent-memory/getting-started-guide/creating-development-environments/linux-environments/linux-memmap) to retain the ISO

The key point lies in preserving the respective ISO file without further storage throughout the kexec. That is achieved by using a persistent memory driver, which creates a pmem device at `/dev/pmem[N]` when booting Linux with the `memmap` parameter.

Note the `memmap` kernel parameter. It is crucial for the kernel
to have pmem enabled to create a block device that contains the ISO file.
The pmem device must be large enough to contain the ISO, and the next kernel needs to know that the device exists and contains its respective ISO file.


### Main Issues
One caveat is that both our webboot kernel, as well as the kernel we kexec into, need support for pmem. See [below](#supported-operating-systems-(Legacy)) for details on OS distribution support and how the kernel needs to be configured.

A second issue is selecting the size options. The Linux system
that starts first needs enough memory to work with, and the pmem device needs to be large enough to hold the ISO. In addition, the addresses provided to `memmap` might segment memory in a way that is difficult to allocate.

Finally, downloading the ISO every time we run `webboot` is a major inconvenience. This issue led us to consider adding a local ISO cache.


### Supported Operating Systems (Legacy)

- [x] TinyCore Linux (remastering the ISO or reusing the webboot kernel for it)
- [x] Arch Linux (PoC for a remastered ISO)
- [x] SystemRescueCd (PoC for a remastered ISO)
- [x] Manjaro (PoC for a remastered ISO)
- [x] Fedora
- [x] openSUSE
- [ ] Debian
- [ ] Ubuntu

### Issue: ISO structure

The respective ISOs of the following distros have pmem as a module in their
squashfs (`lib/modules/*/kernel/drivers/nvdimm/`). They would need to either
have the module in their initramfs already or even built into their kernel.
Otherwise, when we kexec into the respective kernel, we lose the ISO.

The following table lists how the distros structure the ISOs.

| distro         | base dir                      | kernel      | initrd          | squashfs                              |
| -------------- | ----------------------------- | ----------- | --------------- | ------------------------------------- |
| TinyCore       | `/boot`                       | `vmlinuz64` | `corepure64.gz` |  N/A, uses `/cde` directory           |
| Arch Linux     | `/arch/boot/x86_64`           | `vmlinuz`   | `archiso.img`   | `/arch/x86_64/airootfs.sfs`           |
| SystemRescueCd | `/systemrescuecd/boot/x86_64` | `vmlinuz`   | `sysresccd.img` | `/systemrescuecd/x86_64/airootfs.sfs` |
| Fedora         | `/isolinux/`                  | `vmlinuz`   | `initrd.img`    | `/LiveOS/squashfs.img`                |
| openSUSE       | `/boot/x86_64/loader`         | `linux`     | `initrd`        | `/LiveOS/squashfs.img`                |
| Debian         | TODO                          | TODO        | TODO            | TODO                                  |
| Ubuntu         | `/casper`                     | `vmlinuz`   | `initrd`        | `filesystem.squashfs`                 |

For more details, see [distros.md](distros.md).

#### Solutions

1) As a hackaround, we could mount the squashfs within webboot already, copy out
the pmem modules, recreate the initramfs with the pmem modules in addition, and
then boot into that.

2) A much easier way would be to ask the distributors to include the modules for
pmem support already in their initramfs, i.e., the nvdimm drivers, or build them
into the kernel.

The modules to include are `nd_e820`, `nd_btt` and `nd_pmem`.
See also https://cateee.net/lkddb/web-lkddb/X86_PMEM_LEGACY.html
and https://cateee.net/lkddb/web-lkddb/BLK_DEV_PMEM.html.

For the latter option, their config would need to include the following:

```
CONFIG_X86_PMEM_LEGACY_DEVICE=y
CONFIG_X86_PMEM_LEGACY=y
CONFIG_BLK_DEV_PMEM=y
CONFIG_ARCH_HAS_PMEM_API=y
```

3) A third option would be rebuilding the respective distro's kernel on our side
with the options as listed above as a PoC to show them that it works. Then we
could upstream patches.

For a start, the first iteration is a remastered ISO for TinyCore, with a
modified kernel as per 3). The result is stored in
[a separate repository](https://github.com/u-root/webboot-distro/).

For Arch, here are the [full steps](docs/remaster-arch-iso.md), also applicable
to SystemRescueCd, just with different paths as per the table above.

For Manjaro, which is based on Arch, the process is a bit different.
Read the [steps and findings](docs/remaster-manjaro-iso.md) carefully.
