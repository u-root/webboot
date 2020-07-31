package main

import (
	"flag"
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

var (
	v       = flag.Bool("verbose", false, "Verbose output")
	verbose = func(string, ...interface{}) {}
)

// exec downloads the iso and boot it.
func (i *ISO) exec() error {
	// todo: boot the iso
	verbose("ISO is at %s\n", i.path)
	return nil
}

// exec lets user input the name of the iso they want
// if this iso is existed in the bookmark, use it's url
// elsewise ask for a download link
func (d *DownloadOption) exec(uiEvents <-chan ui.Event) (menu.Entry, error) {
	validIsoName := func(input string) (string, string, bool) {
		re := regexp.MustCompile(`[\w]+.iso`)
		if re.Match([]byte(input)) {
			return input, "", true
		}
		return "", "File name should only contain [a-zA-Z0-9_], and should end in .iso", false
	}
	filename, err := menu.NewInputWindow("Enter ISO name", validIsoName, uiEvents)
	if err != nil {
		return nil, err
	}

	link, ok := bookmarks[filename]
	if !ok {
		if link, err = menu.NewInputWindow("Enter URL:", menu.AlwaysValid, uiEvents); err != nil {
			return nil, err
		}
	}

	fpath := filepath.Join("/tmp", filename)
	if err = download(link, fpath); err != nil {
		return nil, err
	}

	var iso *ISO = &ISO{
		label: filename,
		path:  fpath,
	}

	return iso, nil
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
	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
}

func main() {
	flag.Parse()
	log.Println(*v)
	if *v {
		verbose = log.Printf
	}

	entries := []menu.Entry{
		&DownloadOption{},
	}

	entry, err := menu.DisplayMenu("Webboot", "Choose an ISO:", entries, ui.PollEvents())
	if err != nil {
		log.Fatal(err)
	}

	// check the chosen entry of each level
	// and call it's exec() to get the next level's chosen entry.
	// repeat this process until there is no next level
	for entry != nil {
		if downloadOption, ok := entry.(*DownloadOption); ok {
			if entry, err = downloadOption.exec(ui.PollEvents()); err != nil {
				log.Fatal(err)
			}
			continue
		}
		if iso, ok := entry.(*ISO); ok {
			if err = iso.exec(); err != nil {
				log.Fatal(err)
			}
			entry = nil
			continue
		}
		log.Fatalf("Meet an unknow type %T!\n", entry)
	}
}
