package main

import (
	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

const (
	tcURL    = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	wbtcpURL = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso"
)

// bookmark record the list of iso webboot allow user to download
var (
	bookmark = map[string]([]*BookMarkISO){
		"bookmark Tinycore1": []*BookMarkISO{
			&BookMarkISO{
				url:   tcURL,
				label: "Download Tinycore v10.1",
				name:  "TinyCorePure64-10.1.iso",
			},
		},
		"bookmark Tinycore2": []*BookMarkISO{
			&BookMarkISO{
				url:   wbtcpURL,
				label: "Webboot Tinycorepure",
				name:  "TinyCorePure64.iso",
			},
		},
	}
)

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
	group string
	// todo: information parsed from config file
	// configs: []*boot.LinuxImage
}

// Label is the string this iso displays in the menu page.
func (c *CachedISO) Label() string {
	return c.label
}

// DownloadByBookmark option will let user to choose a group
// of the download bookmark
type DownloadByBookmark struct {
	uiEvents <-chan ui.Event
}

// Label is the string this option displays in the menu page.
func (d *DownloadByBookmark) Label() string {
	return "Download by bookmark"
}

// DownloadByLink option will ask user for the name and download link
// of iso then download and boot it
type DownloadByLink struct {
	uiEvents <-chan ui.Event
}

// Label is the string this option displays in the menu page.
func (d *DownloadByLink) Label() string {
	return "Download by link"
}

// InstallCachedISO option will let user to choose a group
// of the cached iso
type InstallCachedISO struct {
	uiEvents  <-chan ui.Event
	cachedISO []*CachedISO
}

// Label is the string this options displays in the menu page.
func (i *InstallCachedISO) Label() string {
	return "Install cached ISO"
}

// Group option display cached isos or bookmarks
// under a certain group
type Group struct {
	name     string
	uiEvents <-chan ui.Event
	entries  []menu.Entry
}

// Label is the string this option displays in the menu page.
func (g *Group) Label() string {
	return g.name
}
