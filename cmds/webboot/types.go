package main

import "github.com/u-root/webboot/pkg/menu"

const (
	tcURL    = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	wbtcpURL = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso"
)

// bookmark record the list of iso webboot allow user to download
var bookmarks = map[string]string{
	"TinyCorePure64-10.1.iso": tcURL,
	"TinyCorePure64.iso":      wbtcpURL,
}

var bookmarkList string

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

// BackOption let user back to the upper menu
type BackOption struct {
}

var _ = menu.Entry(&BackOption{})

// Label is the string this iso displays in the menu page.
func (b *BackOption) Label() string {
	return "Go Back"
}
