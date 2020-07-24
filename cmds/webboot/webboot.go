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

// Exec displays a menu of bookmarks
func (d *DownloadByBookmark) Exec() error {
	entries := []menu.Entry{}
	for _, e := range bookmark {
		entries = append(entries, e)
	}

	_, err := menu.DisplayMenu("Bookmarks", "Input your choice", entries, d.uiEvents)
	if err != nil {
		return err
	}
	return nil
}

// Exec asks for link and name, then downloads the iso and boot it.
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

func getCachedIsos(cachedDir string) []*CachedISO {
	isos := []*CachedISO{}
	walkfunc := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() == false && filepath.Ext(path) == ".iso" {
			// todo: mount the iso and parse the config
			var iso *CachedISO = &CachedISO{
				label: info.Name(),
				path:  path,
				// todo: configs
			}
			isos = append(isos, iso)
		}
		return nil
	}
	filepath.Walk(cachedDir, walkfunc)
	return isos
}
