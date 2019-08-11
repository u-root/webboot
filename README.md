# webboot
Tools to let a u-root instance boot signed live distro images over the web

## How to use webboot

### Build initramfs with added webboot commands.
- Run the command `go run buildimage.go` in the home directory of webboot to build initramfs.

### Boot into Qemu 
Tip:Dont use the -nographic option on qemu when booting u-root in qemu.
 `qemu-system-x86_64 -kernel arch/x86/boot/bzImage  -append 'console=ttyS0 console=tty1' -initrd /tmp/initramfs.linux_amd64.cpio -enable-kvm -m 20G  -device virtio-rng-pci`

### Run the webboot command in Qemu

 `webboot distrubution`

- distrubution a defines a supported operating system distribution.
- example `webboot tinycore`perfoms a webboot of the TinyCore linux distrubution.

### Supported Operating Systems
- Linux TinyCore

### Roadmap : Systems to be added
- Ubuntu
- ArchLinux

### Building a kernel for webboot

webboot uses a standard kernel which should be fairly portable. This
kernel is also based on a Long Term Stable (LTS) release. It has
worked on every chromebook we've tried.

This kernel is built using a config originally from NiChromeOS (github.com/NiChrome/NiChrome).
Because we are building a standard boot stick formatted with vfat, we don't have the weird
space constraints of NiChrome, so we expect this to diverge over time.

Nevertheless, to keep it all simple, we build it as a non-modular kernel with the wifi
firmware built in. We no longer build the initrd into the kernel, however; that's not needed.

To build, first, be sure you're in a directory you want to be in!
You can actually do the work right here in webboot; the .gitignore
will ignore the two directories you create.

You also need some packages; on ubuntu you can do this:
```
sudo apt install libssl-dev
```

Now get everything and build it:

```
git clone --depth 1 -b v4.12.7 git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git linux
git clone git://git.kernel.org/pub/scm/linux/kernel/git/iwlwifi/linux-firmware.git
cp config-4.12.7 linux/.config
(cd linux && make bzImage)
go run .
lzma -f /tmp/initramfs.linux_amd64.cpio
/usr/bin/qemu-system-x86_64 -machine q35   -m 2048   -kernel linux/arch/x86/boot/bzImage  -initrd /tmp/initramfs.linux_amd64.cpio.lzma  -append consle=tty0
```

You may get an error on that qemu:
```
qemu-system-x86_64: Initialization of device e1000e failed: failed to find romfile "efi-e1000e.rom"
```

if so, you can use this alternative command
```
qemu-system-x86_64 -machine q35 -m 2048 -kernel linux/arch/x86/boot/bzImage -initrd /tmp/initramfs.linux_amd64.cpio.lzma -append 'consle=tty0' -netdev user,id=network0 -device rtl8139,netdev=network0 -serial stdio
```

Also, you can add -enable-kvm to the qemu command lines, if you have kvm, to make things faster.

