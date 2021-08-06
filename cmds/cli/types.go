package main

import (
	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/webboot/pkg/bootiso"
	"github.com/u-root/webboot/pkg/menu"
)

type Distro struct {
	IsoPattern    string
	Checksum      string
	ChecksumType  string
	BootConfig    string
	KernelParams  string
	CustomConfigs []bootiso.Config
	Mirrors       []Mirror
}

type Mirror struct {
	Name string
	Url  string
}

func (m *Mirror) Label() string {
	return m.Name
}

var supportedDistros = map[string]Distro{}

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

// ISO contains information of the iso user wants to boot.
type ISO struct {
	label    string
	path     string
	checksum string
}

var _ = menu.Entry(&ISO{})

// Label is the string this iso displays in the menu page.
func (i *ISO) Label() string {
	return i.label
}

// Config represents one kind of configure of booting an iso.
type Config struct {
	label string
}

var _ = menu.Entry(&Config{})

// Label is the string this iso displays in the menu page.
func (c *Config) Label() string {
	return c.label
}

// DownloadOption lets the user download an iso then boot it.
type DownloadOption struct {
}

var _ = menu.Entry(&DownloadOption{})

// Label is the string this iso displays in the menu page.
func (d *DownloadOption) Label() string {
	return "Download an ISO"
}

// DirOption represents a directory under cache directory.
// It displays its sub-directory or iso files.
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

type BootConfig struct {
	image boot.OSImage
}

func (b *BootConfig) Label() string {
	return b.image.Label()
}
