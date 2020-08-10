package main

const (
	tcURL    = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	wbtcpURL = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso"
)

// bookmark record the list of iso webboot allow user to download
var bookmarks = map[string]string{
	"TinyCorePure64-10.1.iso": tcURL,
	"TinyCorePure64.iso":      wbtcpURL,
}

// ISO contains information of the iso user want to boot
type ISO struct {
	label string
	path  string
}

// Label is the string this iso displays in the menu page.
// expected to implement Entry interface
func (i *ISO) Label() string {
	return i.label
}

// Config represents one kind of configure of booting an iso
type Config struct {
	label string
}

// Label is the string this iso displays in the menu page.
func (c *Config) Label() string {
	return c.label
}

// DownloadOption let user download an iso then boot it
// expected to implement Entry interface
type DownloadOption struct {
}

// Label is the string this iso displays in the menu page.
func (d *DownloadOption) Label() string {
	return "Download an ISO"
}

// UseCacheOption let user boot a cached iso
// expected to implement Entry interface
type UseCacheOption struct {
}

// Label is the string this iso displays in the menu page.
func (u *UseCacheOption) Label() string {
	return "Use Cached ISO"
}

// DirOption represents a directory under cache directory
// DirOption options displays it's sub-directory or iso files
type DirOption struct {
	label string
	path  string
}

// Label is the string this option displays in the menu page.
func (d *DirOption) Label() string {
	return d.label
}

// BackOption let user back to the upper menu
// expected to implement Entry interface
type BackOption struct {
}

// Label is the string this iso displays in the menu page.
func (b *BackOption) Label() string {
	return "Go Back"
}
