package main

import ui "github.com/gizak/termui/v3"

const (
	tcURL    = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	wbtcpURL = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso"
)

// bookmark record the list of iso webboot allow user to download
var bookmark = []*BookMarkISO{
	&BookMarkISO{
		url:   tcURL,
		label: "Download Tinycore v10.1",
		name:  "TinyCorePure64-10.1.iso",
	},
	&BookMarkISO{
		url:   wbtcpURL,
		label: "Webboot Tinycorepure",
		name:  "TinyCorePure64.iso",
	},
}

// BookMarkISO contains information of the iso user want to download
// expected to implement Entry interface
type BookMarkISO struct {
	url      string
	label    string
	name     string
	uiEvents <-chan ui.Event
}

// Label is the string this iso displays in the menu page.
func (b *BookMarkISO) Label() string {
	return b.label
}

// CachedISO contains information of the iso cached in the memory
// expected to implement Entry interface
type CachedISO struct {
	label string
	path  string
	// todo: information parsed from config file
	// configs: []*boot.LinuxImage
}

// Label is the string this iso displays in the menu page.
func (c *CachedISO) Label() string {
	return c.label
}

// DownloadByBookmark is to implement "Download by bookmark" option in the menu
type DownloadByBookmark struct {
	uiEvents <-chan ui.Event
}

// Label is the string this iso displays in the menu page.
func (d *DownloadByBookmark) Label() string {
	return "Download by bookmark"
}

// DownloadByLink is to implement "Download by link" option in the menu
type DownloadByLink struct {
	uiEvents <-chan ui.Event
}

// Label is the string this iso displays in the menu page.
func (d *DownloadByLink) Label() string {
	return "Download by link"
}
