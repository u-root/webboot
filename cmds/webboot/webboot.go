package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/webboot/pkg/bootiso"
	"github.com/u-root/webboot/pkg/menu"
)

var (
	v       = flag.Bool("verbose", false, "Verbose output")
	verbose = func(string, ...interface{}) {}
	dir     = flag.String("dir", "", "Path of cached directory")
	network = flag.Bool("network", true, "If network is false we will not set up network")
	dryRun  = flag.Bool("dry_run", true, "If dry_run is true we won't boot the iso.")
)

var cacheDir = ""

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec(uiEvents <-chan ui.Event, boot bool) error {
	verbose("Intent to boot %s", i.path)
	configs, err := bootiso.ParseConfigFromISO(i.path)
	if err != nil {
		return err
	}
	verbose("Get configs: %+v", configs)
	if !boot {
		return nil
	}

	entries := []menu.Entry{}
	for _, config := range configs {
		entries = append(entries, &Config{label: config.Label()})
	}
	c, err := menu.DisplayMenu("Configs", "Choose an option", entries, uiEvents)
	if err == nil {
		err = bootiso.BootFromPmem(i.path, c.Label())
	}
	return err
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

	// check the directory, if there is a subdirectory, add a DirOption option to next menu
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
	entries = append(entries, &BackOption{})
	return menu.DisplayMenu("Distros", "Choose an option", entries, uiEvents)
}

// getCachedDirectory recognizes the usb stick that contains the cached directory from block devices,
// and return the path of cache dir.
// the cache dir should locate at the root of USB stick  and be named as "Images"
// +-- USB root
// |  +-- Images (<--- the cache directory)
// |     +-- subdirectories or iso files
// ...
func getCachedDirectory() (string, error) {
	blockDevs, err := block.GetBlockDevices()
	if err != nil {
		return "", fmt.Errorf("No available block devices to boot from")
	}

	mountPoints, err := ioutil.TempDir("", "temp-device-")
	if err != nil {
		return "", fmt.Errorf("Cannot create tmpdir: %v", err)
	}

	for _, device := range blockDevs {
		mp, err := mount.TryMount(filepath.Join("/dev/", device.Name), filepath.Join(mountPoints, device.Name), "", mount.ReadOnly)
		if err != nil {
			continue
		}
		cachePath := filepath.Join(mp.Path, "Images")
		if _, err = os.Stat(cachePath); err == nil {
			return cachePath, nil
		}
	}
	return "", fmt.Errorf("Do not find the cache directory: Expected a /Images under at the root of a block device(USB)")
}

func getMainMenu(cacheDir string) menu.Entry {
	entries := []menu.Entry{}
	if cacheDir != "" {
		// UseCacheOption is a special DirOption represents the root of cache dir
		entries = append(entries, &DirOption{label: "Use Cached ISO", path: cacheDir})
	}
	entries = append(entries, &DownloadOption{})

	entry, err := menu.DisplayMenu("Webboot", "Choose an option:", entries, ui.PollEvents())
	if err != nil {
		log.Fatal(err)
	}
	return entry
}

func goBackDir(currentPath string) menu.Entry {
	backTo := filepath.Dir(currentPath)
	entry := &DirOption{path: backTo}
	return entry
}

func main() {
	flag.Parse()
	if *v {
		verbose = log.Printf
	}
	cacheDir = *dir
	if cacheDir != "" {
		// call filepath.Clean to make sure the format of path is consistent
		// we should check the cacheDir != "" before call filepath.Clean, because filepath.Clean("") = "."
		cacheDir = filepath.Clean(cacheDir)
	} else {
		if cachePath, err := getCachedDirectory(); err != nil {
			verbose("Fail to find the USB stick: %+v", err)
		} else {
			cacheDir = cachePath
		}
	}
	entry := getMainMenu(cacheDir)

	var err error
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
			if err = entry.(*ISO).exec(ui.PollEvents(), *dryRun); err != nil {
				log.Fatalf("ISO option failed:%v", err)
			}
			entry = nil
		case *DirOption:
			dirOption := entry.(*DirOption)
			if entry, err = dirOption.exec(ui.PollEvents()); err != nil {
				log.Fatalf("Directory option failed:%v", err)
			}
			if _, ok := entry.(*BackOption); ok {
				// if dirOption.path == cacheDir means current dir is the root of cache dir
				// and it should go back to the main menu.
				if dirOption.path == cacheDir {
					entry = getMainMenu(cacheDir)
					break
				}
				entry = goBackDir(dirOption.path)
			}
		default:
			log.Fatalf("Unknown type %T!\n", entry)
		}
	}
}
