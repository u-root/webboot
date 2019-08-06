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
