package main

import (
	"flag"
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

var (
	v       = flag.Bool("verbose", false, "Verbose output")
	verbose = func(string, ...interface{}) {}
)

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec() error {
	// todo: boot the iso
	verbose("ISO is at %s\n", i.path)
	return nil
}

// DownloadOption's exec lets user input the name of the iso they want
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

	return &ISO{label: filename, path: fpath}, nil
}

// DirOption's exec displays subdirectory or cached isos under the path directory
func (d *DirOption) exec(uiEvents <-chan ui.Event) (menu.Entry, error) {
	entries := []menu.Entry{}
	readerInfos, err := ioutil.ReadDir(d.path)
	if err != nil {
		return nil, err
	}

	// check the directory, if there is a subdirectory, add another DirOption option
	// if there is iso file, add an ISO option
	for _, info := range readerInfos {
		if info.IsDir() {
			entries = append(entries, &DirOption{
				label: info.Name(),
				path:  filepath.Join(d.path, info.Name()),
			})
		} else if filepath.Ext(info.Name()) == ".iso" {
			iso := &ISO{
				path:  filepath.Join(d.path, info.Name()),
				label: info.Name(),
			}
			entries = append(entries, iso)
		}
	}
	return menu.DisplayMenu("Distros", "Choose an option", entries, uiEvents)
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
// todo: add a download progress bar
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
	if *v {
		verbose = log.Printf
	}

	entries := []menu.Entry{
		&DirOption{
			label: "Use Cached ISO",
			// todo: replace ./testdata with cache directory
			path: "./testdata",
		},
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
		switch entry.(type) {
		case *DownloadOption:
			if entry, err = entry.(*DownloadOption).exec(ui.PollEvents()); err != nil {
				log.Fatal(err)
			}
		case *ISO:
			if err = entry.(*ISO).exec(); err != nil {
				log.Fatal(err)
			}
			entry = nil
		case *DirOption:
			if entry, err = entry.(*DirOption).exec(ui.PollEvents()); err != nil {
				log.Fatal(err)
			}
		default:
			log.Fatalf("Meet an unknow type %T!\n", entry)
		}
	}
}
