package main

import (
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/bootiso"
	"github.com/u-root/webboot/pkg/menu"
)

var (
	v       = flag.Bool("verbose", false, "Verbose output")
	verbose = func(string, ...interface{}) {}
	dir     = flag.String("dir", "", "Path of cached directory")
	network = flag.Bool("network", true, "Should or not set up network")
	boot    = flag.Bool("boot", true, "Should or not boot the iso. May trun it off for test.")
)

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec(uiEvents <-chan ui.Event, boot bool) error {
	verbose("Intent to boot %s", i.path)
	configs, err := bootiso.ParseConfigFromISO(i.path)
	if err != nil {
		return err
	}
	verbose("Get configs: %+v", configs)
	if boot {
		entries := []menu.Entry{}
		for _, config := range configs {
			entries = append(entries, &Config{label: config.Label()})
		}
		if c, err := menu.DisplayMenu("Configs", "Choose an option", entries, uiEvents); err == nil {
			if err := bootiso.BootFromPmem(i.path, c.Label()); err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	return nil
}

// DownloadOption's exec lets user input the name of the iso they want
// if this iso is existed in the bookmark, use it's url
// elsewise ask for a download link
func (d *DownloadOption) exec(uiEvents <-chan ui.Event, network bool) (menu.Entry, error) {
	if network {
		for {
			ok, err := setUpNetwork(uiEvents)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
		}
	}
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

	entries := []menu.Entry{
		&DownloadOption{},
	}

	if *dir == "" {
		mp, err := getCachedDirectory()
		if err == nil {
			entries = append(entries, &DirOption{
				label: "Use Cached ISO",
				path:  filepath.Join(mp.Path, "Image"),
			})
		}
	} else {
		entries = append(entries, &DirOption{
			label: "Use Cached ISO",
			path:  *dir,
		})
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
			if entry, err = entry.(*DownloadOption).exec(ui.PollEvents(), *network); err != nil {
				log.Fatalf("Download option failed:%v", err)
			}
		case *ISO:
			if err = entry.(*ISO).exec(ui.PollEvents(), *boot); err != nil {
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
