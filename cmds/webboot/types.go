package main

import (
	"fmt"

	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/webboot/pkg/menu"
	"github.com/u-root/webboot/pkg/wifi"
)

type Distro struct {
	url          string
	isoPattern   string
	bootConfig   string
	kernelParams string
}

var supportedDistros = map[string]Distro{
	"Fedora": Distro{
		url:          "https://download.fedoraproject.org/pub/fedora/linux/releases/32/Workstation/x86_64/iso/Fedora-Workstation-Live-x86_64-32-1.6.iso",
		isoPattern:   "^Fedora-.+",
		bootConfig:   "grub",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"Linux Mint": Distro{
		url:          "http://mirrors.kernel.org/linuxmint/stable/20/linuxmint-20-cinnamon-64bit.iso",
		isoPattern:   "^linuxmint-.+",
		bootConfig:   "grub",
		kernelParams: "iso-scan/filename={{.IsoPath}}",
	},
	"Tinycore": Distro{
		url:          "http://tinycorelinux.net/11.x/x86_64/release/TinyCorePure64-11.1.iso",
		isoPattern:   ".*CorePure64-.+",
		bootConfig:   "syslinux",
		kernelParams: "iso=UUID={{.UUID}}{{.IsoPath}}",
	},
	"Ubuntu": Distro{
		url:          "https://releases.ubuntu.com/20.04.1/ubuntu-20.04.1-desktop-amd64.iso",
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
