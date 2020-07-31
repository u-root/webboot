package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

// Exec downloads the iso and boot it.
func (b *BookMarkISO) Exec() error {
	fPath := filepath.Join("/tmp", b.label)
	if err := download(b.url, fPath); err != nil {
		return err
	}
	// todo: boot the iso
	log.Printf("ISO is downloaded at %s", fPath)
	return nil
}

// Exec boots the cached iso.
func (c *CachedISO) Exec() error {
	// todo: boot the iso
	log.Printf("ISO is cached at %s", c.path)
	return nil
}

// Exec ask for the name and download link of iso then download and boot it
func (d *DownloadByLink) Exec() error {
	link, err := menu.NewInputWindow("Enter URL:", menu.AlwaysValid, d.uiEvents)
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
	filename, err := menu.NewInputWindow("Enter ISO name", validIsoName, d.uiEvents)
	if err != nil {
		return err
	}
	fPath := filepath.Join("/tmp", filename)
	if err := download(link, fPath); err != nil {
		return err
	}
	// todo: boot the iso
	log.Printf("ISO is downloaded at %s", fPath)
	return nil
}

// Exec displays subdirectory or cached isos under the path directory
func (g *DirGroup) Exec() error {
	entries := []menu.Entry{}
	readerInfos, err := ioutil.ReadDir(g.path)
	if err != nil {
		return err
	}

	// check the directory, if there is a subdirectory, add another DirGroup option
	// if there is iso file, add a cachedISO option
	for _, info := range readerInfos {
		if info.IsDir() {
			entries = append(entries, &DirGroup{
				label:    info.Name(),
				path:     filepath.Join(g.path, info.Name()),
				uiEvents: g.uiEvents,
			})
		} else if filepath.Ext(info.Name()) == ".iso" {
			iso := &CachedISO{
				path:  filepath.Join(g.path, info.Name()),
				label: info.Name(),
			}
			entries = append(entries, iso)
		}
	}
	_, err = menu.DisplayMenu("Distros", "Choose an option", entries, g.uiEvents)
	return err
}

// Exec displays the next level of bookmarks
func (g *BookmarkGroup) Exec() error {
	nextLevel := []menu.Entry{}
	for _, e := range g.entries {
		if b, ok := e.(*BookmarkGroup); ok {
			b.uiEvents = g.uiEvents
			nextLevel = append(nextLevel, b)
		} else {
			nextLevel = append(nextLevel, e)
		}

	}
	_, err := menu.DisplayMenu("Bookmarks", "Choose an option", nextLevel, g.uiEvents)
	return err
}

// linkOpen sets up a http session to the URL
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

// download downloads a file from URL and save it as fPath
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
		return fmt.Errorf("Fail to copy iso to a persistent memory device: %v", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("Fail to  close %s: %v", fPath, err)
	}
	return nil
}

// getMainMenu makes a hierarchy menu:
// The fisrt step is to choose from: 0.Install cached ISO;1.Download ISO by bookmark; 2.Download ISO by link
// Then the menu will display nested levels of directories or bookmarks until an ISO is chosen
// todo:After choose an iso, will display all available booting methods(different boot cmd parsed from config)
func getMainMenu(cachedDir string, uiEvents <-chan ui.Event) error {

	entries := []menu.Entry{}

	// todo: 1.check all block devices to find usb stick with /image directory
	//		 2.replace the cachedDir parameter

	entries = append(entries,
		&DirGroup{uiEvents: uiEvents, path: cachedDir, label: "Install cached ISO"},
		&BookmarkGroup{uiEvents: uiEvents, entries: bookmarks, label: "Download ISO by Bookmark"},
		&DownloadByLink{uiEvents: uiEvents})
	_, err := menu.DisplayMenu("Webboot", "Choose a boot method", entries, uiEvents)
	return err
}
