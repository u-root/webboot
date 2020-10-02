package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	ui "github.com/gizak/termui/v3"
	Boot "github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/webboot/pkg/bootiso"
	"github.com/u-root/webboot/pkg/menu"
)

var (
	v         = flag.Bool("verbose", false, "Verbose output")
	verbose   = func(string, ...interface{}) {}
	dir       = flag.String("dir", "", "Path of cached directory")
	network   = flag.Bool("network", true, "If network is false we will not set up network")
	dryRun    = flag.Bool("dryrun", false, "If dry_run is true we won't boot the iso.")
	cacheDev  CacheDevice
	logBuffer bytes.Buffer
)

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec(uiEvents <-chan ui.Event, boot bool) error {
	verbose("Intent to boot %s", i.path)

	distroName := inferIsoType(path.Base(i.path))
	distro, ok := supportedDistros[distroName]

	if !ok {
		// Could not infer ISO type based on filename
		// Prompt user to identify the ISO's type
		entries := supportedDistroEntries()
		entry, err := menu.PromptMenuEntry("ISO Type", "Select the closest distribution:", entries, uiEvents)
		if err != nil {
			return err
		}

		distro = supportedDistros[entry.Label()]
	}

	var configs []Boot.OSImage
	if distro.bootConfig != "" {
		parsedConfigs, err := bootiso.ParseConfigFromISO(i.path, distro.bootConfig)
		if err != nil {
			return err
		}

		configs = append(configs, parsedConfigs...)
	}

	if len(distro.customConfigs) != 0 {
		customConfigs, err := bootiso.LoadCustomConfigs(i.path, distro.customConfigs)
		if err != nil {
			return err
		}

		configs = append(configs, customConfigs...)
	}

	if len(configs) == 0 {
		return fmt.Errorf("No valid configs were found.")
	}

	verbose("Get configs: %+v", configs)
	if !boot {
		return fmt.Errorf("Booting is disabled (see --dryrun flag).")
	}

	entries := []menu.Entry{}
	for _, config := range configs {
		entries = append(entries, &BootConfig{config})
	}

	entry, err := menu.PromptMenuEntry("Configs", "Choose an option", entries, uiEvents)
	if err != nil {
		return err
	}

	config, ok := entry.(*BootConfig)
	if !ok {
		return fmt.Errorf("Could not convert selection to a boot image.")
	}

	if err == nil {
		cacheDev.IsoPath = strings.ReplaceAll(i.path, cacheDev.MountPoint, "")
		paramTemplate, err := template.New("template").Parse(distro.kernelParams)
		if err != nil {
			return err
		}

		var kernelParams bytes.Buffer
		if err = paramTemplate.Execute(&kernelParams, cacheDev); err != nil {
			return err
		}

		err = bootiso.BootCachedISO(config.image, kernelParams.String())
	}

	// If kexec succeeds, we should not arrive here
	return err
}

// DownloadOption's exec lets user input the name of the iso they want
// if this iso is existed in the bookmark, use it's url
// elsewise ask for a download link
func (d *DownloadOption) exec(uiEvents <-chan ui.Event, network bool, cacheDir string) (menu.Entry, error) {
	progress := menu.NewProgress("Testing network connection", true)
	activeConnection := connected()
	progress.Close()

	if network && !activeConnection {
		if err := setupNetwork(uiEvents); err != nil {
			return nil, err
		}
	}

	entries := supportedDistroEntries()
	customLabel := "Other Distro"
	entries = append(entries, &Config{customLabel})
	entry, err := menu.PromptMenuEntry("Linux Distros", "Choose an option:", entries, uiEvents)
	if err != nil {
		return nil, err
	}

	var link string
	if entry.Label() == customLabel {
		link, err = menu.PromptTextInput("Enter URL:", validURL, uiEvents)
		if err != nil {
			return nil, err
		}
	} else {
		distro := supportedDistros[entry.Label()]
		link = distro.url
	}
	filename := path.Base(link)

	// If the cachedir is not find, downloaded the iso to /tmp, else create a Downloaded dir in the cache dir.
	var fpath string
	if cacheDir == "" {
		fpath = filepath.Join(os.TempDir(), filename)
	} else {
		downloadDir := filepath.Join(cacheDir, "Downloaded")
		if err = os.MkdirAll(downloadDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("Fail to create the downloaded dir :%v", err)
		}
		fpath = filepath.Join(downloadDir, filename)
	}

	if fileExists(fpath) {
		redownload, err := checkDownloadRequired(filename, fpath, uiEvents)
		if err != nil {
			return nil, err
		} else if !redownload { // return to main menu
			return nil, menu.BackRequest
		}
	}

	if err = download(link, fpath, uiEvents); err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("Download was canceled.")
		} else {
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

	return menu.PromptMenuEntry("Distros", "Choose an option:", entries, uiEvents)
}

// checkDownloadRequired is called if the user asks to download an ISO,
// but the file already exists. Determine if we should remove the existing files
// and redownload, or go back to the main menu so user can select the cached option
func checkDownloadRequired(isoName, isoPath string, uiEvents <-chan ui.Event) (bool, error) {
	var msg string
	checksumPath, checksumType := checksumInfo(isoPath)

	//  There are 3 cases to check for:
	//   1. ISO exists, but there is no checksum file
	//   2. ISO and checksum file exist, but checksum does not match ISO
	//   3. ISO and checksum file exist, and checksum matches the ISO
	if checksumPath == "" {
		msg = fmt.Sprintf("ISO already exists, but no checksum file was found. Would you like to erase %s and redownload?", isoName)
	} else {
		valid, err := bootiso.VerifyChecksum(isoPath, checksumPath, checksumType)
		if err != nil {
			return false, err
		} else if !valid {
			msg = fmt.Sprintf("ISO already exists, but the checksum is invalid. Would you like to erase %s and redownload?", isoName)
		} else {
			msg = fmt.Sprintf("Valid copy of %s already exists. Are you sure you want to redownload and overwrite the file?", isoName)
		}
	}

	//  In any case, give the user 2 options:
	// 	 1. Erase the existing files and redownload
	// 	 2. Keep the existing file and go back to the main menu
	remove, err := menu.PromptConfirmation(msg, uiEvents)
	if err != nil {
		return false, err
	} else if remove {
		os.Remove(isoPath)
		if checksumPath != "" {
			os.Remove(checksumPath)
		}
		return true, nil
	} else {
		return false, nil
	}
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
		mp, err := mount.TryMount(filepath.Join("/dev/", device.Name), filepath.Join(mountPoints, device.Name), "", 0)
		if err != nil {
			continue
		}
		cachePath := filepath.Join(mp.Path, "Images")
		if _, err = os.Stat(cachePath); err == nil {
			cacheDev = NewCacheDevice(device, mp.Path)
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

	for {
		// Display the main menu until user makes a valid choice or
		// they encounter an error that's not menu.BackRequest
		entry, err := menu.PromptMenuEntry("Webboot", "Choose an option:", entries, ui.PollEvents())
		if err != nil && err != menu.BackRequest {
			log.Fatal(err)
		} else if entry != nil {
			return entry
		}
	}
}

func handleError(err error) {
	if err == menu.ExitRequest {
		menu.Close()
		os.Exit(0)
	} else if err == menu.BackRequest {
		return
	}

	errorText := err.Error() + "\n" + logBuffer.String() + wifiStdout.String() + wifiStderr.String()
	menu.DisplayResult(strings.Split(errorText, "\n"), ui.PollEvents())

	logBuffer.Reset()
	wifiStdout.Reset()
	wifiStderr.Reset()
}

func main() {

	flag.Parse()
	if *v {
		verbose = log.Printf
	}

	cacheDir := *dir
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

	if err := menu.Init(); err != nil {
		log.Fatalf(err.Error())
	}

	entry := getMainMenu(cacheDir)

	// Buffer the log output, else it might overlap with the menu
	log.SetOutput(&logBuffer)

	// check the chosen entry of each level
	// and call it's exec() to get the next level's chosen entry.
	// repeat this process until there is no next level
	var err error
	for entry != nil {
		switch entry.(type) {
		case *DownloadOption:
			if entry, err = entry.(*DownloadOption).exec(ui.PollEvents(), *network, cacheDir); err != nil {
				handleError(err)
				entry = getMainMenu(cacheDir)
			}
		case *ISO:
			if err = entry.(*ISO).exec(ui.PollEvents(), !*dryRun); err != nil {
				handleError(err)
				entry = getMainMenu(cacheDir)
			}
		case *DirOption:
			dirOption := entry.(*DirOption)
			if entry, err = dirOption.exec(ui.PollEvents()); err != nil {
				// Check if user requested to go back from a cache subdirectory,
				// so we can send them to a DirOption for the parent directory
				if err == menu.BackRequest && dirOption.path != cacheDir {
					entry = &DirOption{path: filepath.Dir(dirOption.path)}
				} else {
					// Otherwise they either requested to go back from the
					// cache root, so we can send them to main menu,
					// or they encountered an error
					handleError(err)
					entry = getMainMenu(cacheDir)
				}
			}
		default:
			handleError(fmt.Errorf("Unknown menu type %T!\n", entry))
			entry = getMainMenu(cacheDir)
		}
	}
}
