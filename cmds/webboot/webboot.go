package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

const (
	downloadByLinkLabel = "Download by link"
	tcURL               = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	wbtcpURL            = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso"
)

// bookmark record the list of iso webboot allow user to download
var bookmark = []*BookMarkISO{
	&BookMarkISO{
		isDefault: false,
		url:       tcURL,
		label:     "Download Tinycore v10.1",
		isoName:   "TinyCorePure64-10.1.iso",
	},
	&BookMarkISO{
		isDefault: false,
		url:       wbtcpURL,
		label:     "Webboot Tinycorepure",
		isoName:   "TinyCorePure64.iso",
	},
}

// BookMarkISO contains information of the iso user want to download
// expected to implement Entry interface
type BookMarkISO struct {
	isDefault bool
	url       string
	label     string
	isoName   string
}

// Label is the string this iso displays in the menu page.
func (b *BookMarkISO) Label() string {
	return b.label
}

// IsDefault is for mark whether this iso is a default choice.
func (b *BookMarkISO) IsDefault() bool {
	return b.isDefault
}

// Exec performs the following process after the entry is chosen
func (b *BookMarkISO) Exec(_ <-chan ui.Event) error {
	link := b.url
	filename := b.isoName
	fPath := filepath.Join("/tmp", filename)
	if err := download(link, fPath); err != nil {
		return err
	}
	// todo: boot the iso
	return nil
}

// CachedISO contains information of the iso cached in the memory
// expected to implement Entry interface
type CachedISO struct {
	isDefault bool
	label     string
	path      string
	isoName   string
	// todo: information parsed from config file
}

// DownloadByBookmark is to implement "Download by bookmark" option in the menu
type DownloadByBookmark struct{}

// Label is the string this iso displays in the menu page.
func (d *DownloadByBookmark) Label() string {
	return "Download by link"
}

// IsDefault is for mark whether this iso is a default choice.
// assume that DownloadByLink will not be a default option.
func (d *DownloadByBookmark) IsDefault() bool {
	return false
}

// Exec performs the following process after the entry is chosen
func (d *DownloadByBookmark) Exec(uiEvents <-chan ui.Event) error {
	entries := []menu.Entry{}
	for _, e := range bookmark {
		entries = append(entries, e)
	}

	chosen, err := menu.DisplayMenu("Bookmarks", "Input your choice", entries, uiEvents)
	if err != nil {
		return err
	}

	return chosen.Exec(uiEvents)
	// todo: boot the iso
}

// DownloadByLink is to implement "Download by link" option in the menu
type DownloadByLink struct{}

// Label is the string this iso displays in the menu page.
func (d *DownloadByLink) Label() string {
	return "Download by link"
}

// IsDefault is for mark whether this iso is a default choice.
// assume that DownloadByLink will not be a default option.
func (d *DownloadByLink) IsDefault() bool {
	return false
}

// Exec performs the following process after the entry is chosen
func (d *DownloadByLink) Exec(uiEvents <-chan ui.Event) error {
	link, err := menu.NewInputWindow("Please input the link", menu.AlwaysValid, uiEvents)
	if err != nil {
		return err
	}
	validIsoName := func(input string) (string, string, bool) {
		re := regexp.MustCompile(`[\w]+.iso`)
		if re.Match([]byte(input)) {
			return input, "", true
		}
		return "", "File name should only contain [a-zA-Z0-9_], and should end in .iso", false
	}
	filename, err := menu.NewInputWindow("Input ISO name", validIsoName, uiEvents)
	if err != nil {
		return err
	}
	fPath := filepath.Join("/tmp", filename)
	if err := download(link, fPath); err != nil {
		return err
	}
	// todo: boot the iso
	return nil
}

func linkOpen(URL string) (io.ReadCloser, error) {
	u, err := url.Parse(URL)
	if err != nil {
		log.Fatal(err)
	}
	switch u.Scheme {
	case "file":
		return os.Open(URL[7:])
	case "http", "https":
		resp, err := http.Get(URL)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP Get failed: %v", resp.StatusCode)
		}
		return resp.Body, nil
	}
	return nil, fmt.Errorf("%q: linkopen only supports file://, https://, and http:// schemes", URL)
}

// download will download a file from URL and save it as desDir/filename
func download(URL, fPath string) error {

	isoReader, err := linkOpen(URL)
	if err != nil {
		return err
	}
	defer isoReader.Close()
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, isoReader); err != nil {
		return fmt.Errorf("Error copying to persistent memory device: %v", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("Error closing %s: %v", fPath, err)
	}
	return nil
}
