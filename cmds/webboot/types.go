package main

import (
	"fmt"

	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/webboot/pkg/bootiso"
	"github.com/u-root/webboot/pkg/menu"
	"github.com/u-root/webboot/pkg/wifi"
)

type Distro struct {
	url           string
	isoPattern    string
	checksum      string
	checksumType  string
	bootConfig    string
	kernelParams  string
	customConfigs []bootiso.Config
}

var supportedDistros = map[string]Distro{
	"Arch": Distro{
		url:          "http://mirrors.acm.wpi.edu/archlinux/iso/2021.06.01/archlinux-2021.06.01-x86_64.iso",
		isoPattern:   "^archlinux-.+",
		checksum:     "1bf76d864651cc6454ab273fd3d2226a",
		checksumType: "md5",
		kernelParams: "img_dev=/dev/disk/by-uuid/{{.UUID}} img_loop={{.IsoPath}}",
		customConfigs: []bootiso.Config{
			bootiso.Config{
				Label:      "Default Config",
				KernelPath: "/arch/boot/x86_64/vmlinuz-linux",
				InitrdPath: "/arch/boot/x86_64/archiso.img",
				Cmdline:    "",
			},
		},
	},
	"CentOS 7": Distro{
		url:          "https://sjc.edge.kernel.org/centos/7/isos/x86_64/CentOS-7-x86_64-LiveGNOME-2003.iso",
		isoPattern:   "^CentOS-7.+",
		checksum:     "689531cce9cf484378481ae762fae362791a9be078fda10e4f6977bf8fa71350",
		checksumType: "sha256",
		bootConfig:   "grub",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"CentOS 8": Distro{
		url:          "https://sjc.edge.kernel.org/centos/8.2.2004/isos/x86_64/CentOS-8.2.2004-x86_64-minimal.iso",
		isoPattern:   "^CentOS-8.+",
		checksum:     "http://centos.mirror.lstn.net/8.4.2105/isos/x86_64",
		checksumType: "sha256",
		bootConfig:   "grub",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"Debian": Distro{
		url:          "https://cdimage.debian.org/debian-cd/current-live/amd64/iso-hybrid/debian-live-10.9.0-amd64-xfce.iso",
		isoPattern:   "^debian-.+",
		checksum:     "44e98dfc974e5ade72ebf3cbb9ff06df3aa2c0c0cdc0f30913dbd93983179ff5",
		checksumType: "sha256",
		bootConfig:   "syslinux",
		kernelParams: "findiso={{.IsoPath}}",
	},
	"Fedora": Distro{
		url:          "https://download.fedoraproject.org/pub/fedora/linux/releases/32/Workstation/x86_64/iso/Fedora-Workstation-Live-x86_64-32-1.6.iso",
		isoPattern:   "^Fedora-.+",
		checksum:     "4d0f6653e2e0860c99ffe0ef274a46d875fb85bd2a40cb896dce1ed013566924",
		checksumType: "sha256",
		bootConfig:   "grub",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"Kali": Distro{
		url:          "https://cdimage.kali.org/kali-2020.3/kali-linux-2020.3-live-amd64.iso",
		isoPattern:   "^kali-linux-.+",
		checksum:     "1a0b2ea83f48861dd3f3babd5a2892a14b30a7234c8c9b5013a6507d1401874f",
		checksumType: "sha256",
		bootConfig:   "grub",
		kernelParams: "findiso={{.IsoPath}}",
	},
	"Linux Mint": Distro{
		url:          "http://mirrors.kernel.org/linuxmint/stable/20/linuxmint-20-cinnamon-64bit.iso",
		isoPattern:   "^linuxmint-.+",
		checksum:     "2f6ae466ec9b7c6255e997b82f162ae88bfe640a8df16d3e2f495b6281120af9",
		checksumType: "sha256",
		bootConfig:   "grub",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"Manjaro": Distro{
		url:          "https://download.manjaro.org/xfce/21.0.6/manjaro-xfce-21.0.6-210607-linux510.iso",
		isoPattern:   "^manjaro-.+",
		checksum:     "fab9d1bdd03a7e5daab226ccc8e16ba96a5b07e9",
		checksumType: "sha1",
		kernelParams: "img_dev=/dev/disk/by-uuid/{{.UUID}} img_loop={{.IsoPath}}",
		customConfigs: []bootiso.Config{
			bootiso.Config{
				Label:      "Default Config",
				KernelPath: "/boot/vmlinuz-x86_64",
				InitrdPath: "/boot/initramfs-x86_64.img",
				Cmdline:    "driver=free tz=utc lang=en_US keytable=en",
			},
		},
	},
	"Tinycore": Distro{
		url:          "http://tinycorelinux.net/11.x/x86_64/release/TinyCorePure64-11.1.iso",
		isoPattern:   ".*CorePure64-.+",
		checksum:     "58bc33523ce10e64f56b9a9ec8a77531",
		checksumType: "md5",
		bootConfig:   "syslinux",
		kernelParams: "iso=UUID={{.UUID}}{{.IsoPath}}",
	},
	"LHSCowboys": Distro{
		url:          "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/LHSCowboys.iso",
		isoPattern:   ".*CorePure64-.+",
		bootConfig:   "syslinux",
		kernelParams: "iso=UUID={{.UUID}}{{.IsoPath}}",
	},
	"DHSGaels": Distro{
		url:          "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/DHSGaels.iso",
		isoPattern:   ".*CorePure64-.+",
		bootConfig:   "syslinux",
		kernelParams: "iso=UUID={{.UUID}}{{.IsoPath}}",
	},
	"Ubuntu": Distro{
		url:          "https://releases.ubuntu.com/20.04.1/ubuntu-20.04.1-desktop-amd64.iso",
		isoPattern:   "^ubuntu-.+",
		checksum:     "b45165ed3cd437b9ffad02a2aad22a4ddc69162470e2622982889ce5826f6e3d",
		checksumType: "sha256",
		bootConfig:   "syslinux",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"Ubuntu (Constant.com)": Distro{
		url:          "http://isos.ubuntu.mirror.constant.com/20.04/ubuntu-20.04.2.0-desktop-amd64.iso",
		isoPattern:   "^ubuntu-.+",
		bootConfig:   "syslinux",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
}

type CacheDevice struct {
	Name       string
	UUID       string
	MountPoint string
	IsoPath    string // set after iso is selected
}

func NewCacheDevice(device *block.BlockDev, mountPoint string) CacheDevice {
	return CacheDevice{
		Name:       device.Name,
		UUID:       device.FsUUID,
		MountPoint: mountPoint,
	}
}

// ISO contains information of the iso user want to boot
type ISO struct {
	label string
	path  string
	checksum string
}

var _ = menu.Entry(&ISO{})

// Label is the string this iso displays in the menu page.
func (i *ISO) Label() string {
	return i.label
}

// Config represents one kind of configure of booting an iso
type Config struct {
	label string
}

var _ = menu.Entry(&Config{})

// Label is the string this iso displays in the menu page.
func (c *Config) Label() string {
	return c.label
}

// DownloadOption let user download an iso then boot it
type DownloadOption struct {
}

var _ = menu.Entry(&DownloadOption{})

// Label is the string this iso displays in the menu page.
func (d *DownloadOption) Label() string {
	return "Download an ISO"
}

// DirOption represents a directory under cache directory
// it displays it's sub-directory or iso files
type DirOption struct {
	label string
	path  string
}

var _ = menu.Entry(&DirOption{})

// Label is the string this option displays in the menu page.
func (d *DirOption) Label() string {
	return d.label
}

type Interface struct {
	label string
}

func (i *Interface) Label() string {
	return i.label
}

type Network struct {
	info wifi.Option
}

func (n *Network) Label() string {
	switch n.info.AuthSuite {
	case wifi.NoEnc:
		return fmt.Sprintf("%s: No Passphrase\n", n.info.Essid)
	case wifi.WpaPsk:
		return fmt.Sprintf("%s: WPA-PSK (only passphrase)\n", n.info.Essid)
	case wifi.WpaEap:
		return fmt.Sprintf("%s: WPA-EAP (passphrase and identity)\n", n.info.Essid)
	case wifi.NotSupportedProto:
		return fmt.Sprintf("%s: Not a supported protocol\n", n.info.Essid)
	}
	return "Invalid wifi network."
}

type BootConfig struct {
	image boot.OSImage
}

func (b *BootConfig) Label() string {
	return b.image.Label()
}
