# Distributions of Linux-based systems

Well... https://twitter.com/OrangeCMS/status/1220605490792751104

This file lists details on how the respective distributions patch and configure
their kernels, where to obtain the sources and config files, and the download
URLs for current release ISO images. That helps maintaining and further
developing webboot.

## Arch Linux

- [kernel sources](https://git.archlinux.org/linux.git?signed#tag=v5.4.14-arch1)
- [kernel config](https://git.archlinux.org/svntogit/packages.git/tree/trunk?h=packages/linux)

## Fedora

[Build instructions](https://fedoraproject.org/wiki/Building_a_custom_kernel)
in the wiki are specific to Fedora and not suitable for other systems.
Cloning from the repository and applying the config should just work though.

- [ISO](https://download.fedoraproject.org/pub/fedora/linux/releases/31/Workstation/x86_64/iso/Fedora-Workstation-Live-x86_64-31-1.9.iso)
- [kernel sources](https://src.fedoraproject.org/rpms/kernel/tree/master)
- [stable sources](https://koji.fedoraproject.org/koji/search?terms=kernel-5.5.0-0.rc6.git3.1.fc32&type=build&match=glob)
- [kernel config](https://src.fedoraproject.org/rpms/kernel/raw/master/f/kernel-x86_64-fedora.config)

## openSUSE

Details on the kernel are [in the wiki](https://en.opensuse.org/Kernel), as well
as [instructions](https://en.opensuse.org/openSUSE:Kernel_git).

- [ISO](https://download.opensuse.org/distribution/leap/15.1/iso/openSUSE-Leap-15.1-DVD-x86_64.iso),
  [netboot](https://download.opensuse.org/distribution/leap/15.1/iso/openSUSE-Leap-15.1-NET-x86_64.iso)
- [kernel sources](https://kernel.opensuse.org/cgit/kernel-source/tree/config/x86_64/default?h=openSUSE-15.2)
- kernel config: within sources, see `config/x86_64/default`

## TinyCore

Patched kernel sources, separate patches and config can be found among the
[release sources](http://tinycorelinux.net/10.x/x86/release/src/kernel/).

- [Core ISO](http://tinycorelinux.net/10.x/x86/release/Core-current.iso)
- [CorePlus ISO](http://tinycorelinux.net/10.x/x86/release/CorePlus-current.iso)
- [TinyCore ISO](http://tinycorelinux.net/10.x/x86/release/TinyCore-current.iso)
- [kernel sources](http://tinycorelinux.net/10.x/x86/release/src/kernel/linux-4.19.10-patched.txz)
- [kernel config](http://tinycorelinux.net/10.x/x86/release/src/kernel/config-4.19.10-tinycore)

## Ubuntu

- [kernel sources](http://security.ubuntu.com/ubuntu/pool/main/l/linux-hwe/linux-source-5.0.0_5.0.0-37.40~18.04.1_all.deb)
- kernel config: included in sources, split into parts for common, arch-specific
  and generic/low latency bits in `src/linux-source-5.0.0/debian.hwe/config/`:
  `config.common.ubuntu`, `amd64/config.{common.amd64,flavour.generic}` are
  the relevant files for webboot on x86
