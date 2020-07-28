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
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

// Exec downloads the iso and boot it.
func (b *BookMarkISO) Exec() error {
	fPath := filepath.Join("/tmp", b.name)
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

// Exec let user to choose a group of the bookmark
func (d *DownloadByBookmark) Exec() error {
	groups := []menu.Entry{}
	for key, value := range bookmark {
		entries := []menu.Entry{}
		for _, bm := range value {
			bm.uiEvents = d.uiEvents
			entries = append(entries, bm)
		}
		groups = append(groups, &Group{
			name:     key,
			entries:  entries,
			uiEvents: d.uiEvents,
		})
	}
	_, err := menu.DisplayMenu("Bookmarks", "Choose a group of bookmark", groups, d.uiEvents)
	return err
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

// Exec let user to choose a group of the cached iso
func (i *InstallCachedISO) Exec() error {
	isos := i.cachedISO
	groupedIso := make(map[string]([]*CachedISO))
	for _, iso := range isos {
		groupedIso[(*iso).group] = append(groupedIso[(*iso).group], iso)
	}

	groups := []menu.Entry{}
	for key, value := range groupedIso {
		entries := []menu.Entry{}
		for _, iso := range value {
			entries = append(entries, iso)
		}
		groups = append(groups, &Group{
			name:     key,
			entries:  entries,
			uiEvents: i.uiEvents,
		})
	}
	_, err := menu.DisplayMenu("Groups", "Choose a group of chached iso", groups, i.uiEvents)
	return err
}

// Exec displays cached isos or bookmarks under a certain group
func (g *Group) Exec() error {
	_, err := menu.DisplayMenu("Distro", "Choose a Distro", g.entries, g.uiEvents)
	return err
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

// download will download a file from URL and save it as fPath
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

// will call a function to get all the block devices, then mount it one by one
// Then call this function to all iso under the device and returns them
// Then in the getHierachyMenu function add these iso to the menu
func getCachedIsos(cachedDir string) []*CachedISO {
	isos := []*CachedISO{}
	walkfunc := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() == false && filepath.Ext(path) == ".iso" {
			// todo: mount the iso and parse the config
			dir := filepath.Dir(path)
			var iso *CachedISO = &CachedISO{
				label: info.Name(),
				path:  path,
				group: dir[strings.LastIndex(dir, "/")+1:],
				// todo: configs
			}
			isos = append(isos, iso)
		}
		return nil
	}
	filepath.Walk(cachedDir, walkfunc)
	return isos
}

// getHierachyMenu makes a hierarchy menu:
// level 1: 0.Install cached ISO;1.Download ISO by bookmark; 2.Download ISO by link
// level 2:
// 		Install cached ISO & Download ISO by bookmark: options of groups
//      Download ISO by link: DownloadByLink option
// level 3:
//     	Install cached ISO: cachedISO options belong to the certain group
// 		Download ISO by bookmark: BookMarkISO options belong to the certain group
func getHierachyMenu(cachedDir string, uiEvents <-chan ui.Event) error {

	// todo: remove the cachedDir parameter and check all block devices to find the cached directory
	cachedIsos := getCachedIsos(cachedDir)

	entries := []menu.Entry{}
	installCachedISO := &InstallCachedISO{
		uiEvents:  uiEvents,
		cachedISO: cachedIsos,
	}

	entries = append(entries, installCachedISO, &DownloadByBookmark{uiEvents: uiEvents}, &DownloadByLink{uiEvents: uiEvents})
	_, err := menu.DisplayMenu("Webboot", "Choose a boot method", entries, uiEvents)
	return err
}
