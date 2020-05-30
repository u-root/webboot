# About

webboot offers tools to let a u-root instance boot signed live distro images
over the web.

## Concept

The webboot bootloader works as follows:

1. fetch an OS distro release ISO from the web
2. copy the ISO to memory
3. mount the ISO and copy out the kernel and initrd
4. load the extracted kernel with the initrd
5. kexec that kernel with `memmap` parameters to retain the ISO

The key point lies in preserving the respective ISO file without further storage
throughout the kexec. That is achieved by using a persistent memory driver,
which creates a pmem device at `/dev/pmem[N]` when booting Linux with the
`memmap` parameter.

The caveat is that both our webboot kernel as well as the kernel we kexec into
need support for pmem. See [below](#supported-operating-systems) for details on
OS distribution support and how the kernel needs to be configured.

The second issue is with carefully choosing size options. The Linux system
started first needs enough memory to work with, and the pmem device needs to be
large enough to hold the ISO.

For reference, webboot developers should familiarize themselves with:

- [cpio tutorial](https://www.gnu.org/software/cpio/manual/html_node/Tutorial.html)
- [initrd usage](https://www.kernel.org/doc/html/latest/admin-guide/initrd.html)
- [kernel parameters](https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html)
- the [memmap option](https://docs.pmem.io/persistent-memory/getting-started-guide/creating-development-environments/linux-environments/linux-memmap)

## Usage

### Build initramfs with added webboot commands

Run `go run buildimage.go` in the source directory of webboot to build the
initramfs.

This runs [u-root](https://github.com/u-root/u-root) under the hood. To pass
extra options, such as to include extra files, use the `-u` switch, e.g.,
`go run buildimage.go -u "-files path/to/bzImage:bzImage"` to add a custom
kernel which can be used to test whether kexec works in a small setup. That
saves a lot of time, because a full webboot flow would always need to download
large ISO files, copy them, mount and decompress.

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

To build, first, be sure you're in a directory you want to be in!
You can actually do the work in the webboot root directory because the
`.gitignore` file ignores the two directories you create when following the
instructions here.

#### Prerequisites

You need to have the following packages installed if on Ubuntu:
```sh
sudo apt install libssl-dev build-essential
```

#### Fetching, configuring and compiling the kernel

```sh
git clone --depth 1 -b v4.12.7 \
  git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git linux
git clone \
  git://git.kernel.org/pub/scm/linux/kernel/git/iwlwifi/linux-firmware.git
cp config-4.12.7 linux/.config
(cd linux && make bzImage)
go run .
```

### Testing in QEMU

Tip: Don't use the `-nographic` option for u-root in QEMU as you want to boot
into a graphical interface.

#### Acceleration

If you have KVM in your host system, you can add `-enable-kvm` for speedup.

```sh
qemu-system-x86_64 \
  -enable-kvm \
  -m 20G \
  -kernel linux/arch/x86/boot/bzImage \
  -append 'console=ttyS0 console=tty1 memmap=1G!1G' \
  -initrd /tmp/initramfs.linux_amd64.cpio \
  -device virtio-rng-pci
```

Note the `memmap` kernel parameter for webboot. It is crucial for the kernel
to have pmem enabled to create a block device in memory to mount the ISO of the
next OS to boot into, and for that second OS's kernel to know about it as well
so that it knows where the ISO resides in RAM such that it can pick it up and
load additional files from it, i.e., its root filesystem. The size for `memmap`
needs to be chosen such that the ISO fits into it and it is sufficiently
smaller than the memory assigned to the VM so that the first system has enough
for itself to run.

Refer to
[u-root's documentation](https://github.com/u-root/u-root#testing-in-qemu) for
more details on virtualization.

### The `webboot` command

`webboot [distribution]`

The `distribution` argument defines a supported operating system distribution.
E.g., `webboot tinycore` perfoms a webboot of the TinyCore Linux distribution.

#### Ethernet / QEMU

If you have no Wi-fi but ethernet instead, or you are trying webboot in QEMU
with an emulated ethernet card, you will need to tell webboot to use the
correct interface, e.g., `eth0`:

`webboot -interface eth0 [distribution]`

### Supported Operating Systems

- [x] TinyCore Linux (remastering the ISO or reusing the webboot kernel for it)
- [x] Arch Linux (PoC for a remastered ISO)
- [x] SystemRescueCd (PoC for a remastered ISO)
- [x] Manjaro (PoC for a remastered ISO)
- [ ] Fedora
- [ ] openSUSE
- [ ] Debian
- [ ] Ubuntu

TODO: look into other distros such as http://boot.slitaz.org/en/

#### Issue: ISO structure

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
| openSUSE       | `/boot/x86_64/loader`         | `linux`     | `initrd`        | `/LiveOS/squashfs.img`                |
| Ubuntu         | `/casper`                     | `vmlinuz`   | `initrd`        | `filesystem.squashfs`                 |

For more details, see [distros.md](distros.md).

#### Solutions

1) As a hackaround, we could mount the squashfs within webboot already, copy out
the pmem modules, recreate the initramfs with the pmem modules in addition, and
then boot into that.

2) A much easier way would be to ask the distributors to include the modules for
pmem support already in their initramfs, i.e., the nvdimm drivers, or build them
into the kernel.

The modules to include are `nd_e820`, `nd_pmem` and `nd_pmem`.
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
