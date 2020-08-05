package main

import (
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

var (
	v       = flag.Bool("verbose", false, "Verbose output")
	verbose = func(string, ...interface{}) {}
	dir     = flag.String("dir", "", "Path of cached directory")
)

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec() error {
	// todo: boot the iso
	log.Printf("ISO is at %s\n", i.path)
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
	// if download link is not valid, ask again until the link is rights
	err = download(link, fpath)
	for err != nil {
		err = download(link, fpath)
		if _, derr := menu.DisplayResult([]string{err.Error()}, uiEvents); derr != nil {
			return nil, derr
		}
		if link, err = menu.NewInputWindow("Enter URL:", menu.AlwaysValid, uiEvents); err != nil {
			return nil, err
		}
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

func main() {
	flag.Parse()
	if *v {
		verbose = log.Printf
	}

	cachedDir := *dir

	if cachedDir == "" {
		mp, err := getCachedDirectory()
		if err != nil {
			log.Fatalf("Fail to find the USB stick: %+v", err)
		}
		if mp == nil {
			log.Fatalf("Do not find the cache directory.")
		}
		cachedDir = filepath.Join(mp.Path, "Image")
	}
	entries := []menu.Entry{
		// "Use Cached ISO" option is a special DirGroup Entry
		// which represents the root of the cache directory
		&DirOption{
			label: "Use Cached ISO",
			path:  cachedDir,
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
				log.Fatalf("Download option failed:%v", err)
			}
		case *ISO:
			if err = entry.(*ISO).exec(); err != nil {
				log.Fatalf("ISO option failed:%v", err)
			}
			entry = nil
		case *DirOption:
			if entry, err = entry.(*DirOption).exec(ui.PollEvents()); err != nil {
				log.Fatalf("Directory option failed:%v", err)
			}
		default:
			log.Fatalf("Unknown type %T!\n", entry)
		}
	}
}
