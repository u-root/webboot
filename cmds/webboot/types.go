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
	bookmarks = []menu.Entry{
		&BookmarkGroup{
			label: "Tinycore",
			entries: []menu.Entry{
				&BookmarkGroup{
					label: "TinyCorePure64",
					entries: []menu.Entry{
						&BookMarkISO{
							url:   tcURL,
							label: "TinyCorePure64-10.1.iso",
						},
						&BookMarkISO{
							url:   wbtcpURL,
							label: "Webboot_Tinycorepure.iso",
						},
					},
				},
			},
		},
	}
)

// BookMarkISO contains information of the iso user want to download
// expected to implement Entry interface
type BookMarkISO struct {
	url      string
	label    string
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

// DownloadByLink option asks user for the name and download link
// of iso then download and boot it
type DownloadByLink struct {
	uiEvents <-chan ui.Event
}

// Label is the string this option displays in the menu page.
func (d *DownloadByLink) Label() string {
	return "Download by link"
}

// DirGroup option displays subdirectory or cached isos under a certain directory
// InstallCachedISO option is a special DirGroup option
// which represents the root of the cache directory
type DirGroup struct {
	label    string
	uiEvents <-chan ui.Event
	path     string
}

// Label is the string this option displays in the menu page.
func (g *DirGroup) Label() string {
	return g.label
}

// BookmarkGroup option displays bookmarks under a certain group
// DownloadByBookmark option is a special BookmarkGroup option
// which represents the root level of the bookmarks
type BookmarkGroup struct {
	label    string
	uiEvents <-chan ui.Event
	entries  []menu.Entry
}

// Label is the string this option displays in the menu page.
func (g *BookmarkGroup) Label() string {
	return g.label
}
