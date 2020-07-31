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

// DownloadOption let user download an iso then boot it
// expected to implement Entry interface
type DownloadOption struct {
}

// Label is the string this iso displays in the menu page.
func (d *DownloadOption) Label() string {
	return "Download an ISO"
}
