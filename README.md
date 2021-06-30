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
sudo apt install libssl-dev build-essential wireless-tools kexec-tools bison flex
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
  -append 'console=ttyS0 console=tty1' \
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
- you have a USB stick with MBR partitioning, first partition formatted as VFAT
- you have a directory `/mnt/usb` to mount the partition to
- you have `syslinux` installed (use your package manager)
- you have built a suitable Linux kernel using a config from this repository
- you have built an initramfs that includes webboot, gzip-compressed

To [install](https://wiki.syslinux.org/wiki/index.php?title=Install) syslinux as
a bootloader and configure it, four steps are necessary:

1. write a Volume Boot Record (VBR) to the stick
2. write a Master Boot Record (MBR) to it
3. mark the first partition as bootable
4. copy the config file, Linux kernel and initcpio

Now the following commands would need to be run as root:

```sh
syslinux -i /dev/sdb1
dd bs=440 count=1 conv=notrunc if=/usr/lib/syslinux/bios/mbr.bin of=/dev/sdb
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
