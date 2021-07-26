package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	tmpBuffer bytes.Buffer
)

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec(uiEvents <-chan ui.Event, menus chan<- string, boot bool) error {
	verbose("Intent to boot %s", i.path)

	distroName := inferIsoType(path.Base(i.path), supportedDistros)
	distro, ok := supportedDistros[distroName]

	if !ok {
		// Could not infer ISO type based on filename
		// Prompt user to identify the ISO's type
		entries := supportedDistroEntries()
		entry, err := menu.PromptMenuEntry("ISO Type", "Select the closest distribution:", entries, uiEvents, menus)
		if err != nil {
			return err
		}

		distro = supportedDistros[entry.Label()]
	}

	verbose("Using distro %s with boot config %s", distroName, distro.BootConfig)

	var configs []Boot.OSImage
	if distro.BootConfig != "" {
		parsedConfigs, err := bootiso.ParseConfigFromISO(i.path, distro.BootConfig)
		if err != nil {
			return err
		}

		configs = append(configs, parsedConfigs...)
	}

	if len(distro.CustomConfigs) != 0 {
		CustomConfigs, err := bootiso.LoadCustomConfigs(i.path, distro.CustomConfigs)
		if err != nil {
			return err
		}

		configs = append(configs, CustomConfigs...)
	}

	if len(configs) == 0 {
		return fmt.Errorf("No valid configs were found.")
	}

	verbose("Get configs: %+v", configs)

	entries := []menu.Entry{}
	for _, config := range configs {
		entries = append(entries, &BootConfig{config})
	}

	entry, err := menu.PromptMenuEntry("Configs", "Choose an option", entries, uiEvents, menus)
	if err != nil {
		return err
	}

	config, ok := entry.(*BootConfig)
	if !ok {
		return fmt.Errorf("Could not convert selection to a boot image.")
	}

	if err == nil {
		cacheDev.IsoPath = strings.ReplaceAll(i.path, cacheDev.MountPoint, "")
		paramTemplate, err := template.New("template").Parse(distro.KernelParams)
		if err != nil {
			return err
		}

		var kernelParams bytes.Buffer
		if err = paramTemplate.Execute(&kernelParams, cacheDev); err != nil {
			return err
		}

		if !boot {
			s := fmt.Sprintf("config.image %s, kernelparams.String() %s", config.image, kernelParams.String())
			return fmt.Errorf("Booting is disabled (see --dryrun flag), but otherwise would be [%s].", s)
		}
		err = bootiso.BootCachedISO(config.image, kernelParams.String()+" waitusb=10")
	}

	// If kexec succeeds, we should not arrive here
	if err == nil {
		// TODO: We should know whether we tried using /sbin/kexec.
		err = fmt.Errorf("kexec failed, but gave no error. Consider trying kexec-tools.")
	}

	return err
}

// DownloadOption's exec lets user input the name of the iso they want
// if this iso is existed in the bookmark, use it's url
// elsewise ask for a download link
func (d *DownloadOption) exec(uiEvents <-chan ui.Event, menus chan<- string, network bool, cacheDir string) (menu.Entry, error) {

	entries := supportedDistroEntries()
	customLabel := "Other Distro"
	entries = append(entries, &Config{customLabel})
	entry, err := menu.PromptMenuEntry("Linux Distros", "Choose an option:", entries, uiEvents, menus)
	if err != nil {
		return nil, err
	}
	var link string

	if entry.Label() == customLabel {
		link, err = menu.PromptTextInput("Enter URL:", validURL, uiEvents, menus)
		if err != nil {
			return nil, err
		}
	} else {
		link, _, err = mirrorMenu(entry, uiEvents, menus, link)
		if err != nil {
			return nil, err
		}
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

	if err = download(link, fpath, uiEvents); err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("Download was canceled.")
		} else {
			return nil, err
		}
	}

	menu, err := displayChecksumPrompt(uiEvents, menus, supportedDistros, entry.Label(), fpath)
	if err != nil {
		return nil, err
	} else if menu != nil {
		return menu, nil
	}

	return &ISO{label: filename, path: fpath}, nil
}

// DirOption's exec displays subdirectory or cached isos under the path directory
func (d *DirOption) exec(uiEvents <-chan ui.Event, menus chan<- string) (menu.Entry, error) {
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

	return menu.PromptMenuEntry("Distros", "Choose an option:", entries, uiEvents, menus)
}

// distroData downloads and parses the data in distros.json to a map[string]Distro.
func distroData(uiEvents <-chan ui.Event, menus chan<- string, cacheDir string, jsonLink string) (map[string]Distro, error) {
	// Download the json file.
	var jsonPath string

	if cacheDir == "" {
		jsonPath = filepath.Join(os.TempDir(), "distros.json")
	} else {
		downloadDir := filepath.Join(cacheDir, "Downloaded")
		if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("Fail to create the downloaded dir: %v", err)
		}
		jsonPath = filepath.Join(downloadDir, "distros.json")
	}

	if err := download(jsonLink, jsonPath, uiEvents); err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("JSON file download was canceled.")
		} else {
			entries := []menu.Entry{&Config{label: "Ok"}}
			_, err := menu.PromptMenuEntry("Failed to download JSON file.", "Choose \"Ok\" to proceed using default JSON file.", entries, uiEvents, menus)
			if err != nil {
				return nil, fmt.Errorf("Could not display PromptMenuEntry: %v", err)
			}
			jsonPath = "./distros.json"
		}
	}

	// Parse the json file.
	data, err := ioutil.ReadFile(jsonPath)

	if err != nil {
		return nil, fmt.Errorf("Could not read JSON file: %v\n", err)
	}

	supportedDistros := map[string]Distro{}

	err = json.Unmarshal([]byte(data), &supportedDistros)
	if err != nil {
		return nil, fmt.Errorf("Could not unmarshal JSON file: %v\n", err)
	}

	return supportedDistros, nil
}

// If the chosen distro has a checksum, verify it.
// If the checksum is not correct, prompt the user to choose whether they still want to continue.
func displayChecksumPrompt(uiEvents <-chan ui.Event, menus chan<- string, supportedDistros map[string]Distro, label string, fpath string) (menu.Entry, error) {
	// Check that the distro is supported
	if _, ok := supportedDistros[label]; ok {
		distro := supportedDistros[label]
		// Check that checksum is available
		if distro.Checksum == "" {
			accept, err := menu.PromptConfirmation("This distro does not have a checksum. Proceed anyway?", uiEvents, menus)
			if err != nil {
				return nil, fmt.Errorf("Failed to prompt confirmation: %s", err)
			}
			if !accept {
				// Go back to download menu
				return &DownloadOption{}, nil
			}
		} else if valid, calcChecksum, err := bootiso.VerifyChecksum(fpath, distro.Checksum, distro.ChecksumType); err != nil {
			return nil, fmt.Errorf("Failed to verify checksum: %s", err)
		} else if !valid {
			accept, err := menu.PromptConfirmation(fmt.Sprintf("Checksum was not correct. The correct checksum is %s and the downloaded ISO's checksum is %s. Proceed anyway?",
				distro.Checksum, calcChecksum), uiEvents, menus)
			if err != nil {
				return nil, fmt.Errorf("Failed to prompt confirmation: %s", err)
			}
			if !accept {
				// Go back to download menu
				return &DownloadOption{}, nil
			}
		}
	}
	return nil, nil
}

// mirrorMenu fetches the mirror options of the distro the user selects and displays them in a new menu. Finally, it gets
// the download link of the mirror the user selects.
func mirrorMenu(entry menu.Entry, uiEvents <-chan ui.Event, menus chan<- string, link string) (url string, mirrorNameForTestPurposes string, err error) {
	// Code for after the specific distro has been selected.
	// Looks up the distro.
	distro := supportedDistros[entry.Label()]
	if len(distro.Mirrors) > 0 {
		// Make an array of type menu.Entry to store the mirrors of the
		// particular distro selected. Then, display the mirror options.
		entries := make([]menu.Entry, len(distro.Mirrors))
		for i := range entries {
			entries[i] = &distro.Mirrors[i]
		}
		entry, err = menu.PromptMenuEntry("Available Mirrors", "Choose an option:", entries, uiEvents, menus)
		if err != nil {
			return "", "", err
		}
	}
	// Iterate through the mirrors of the distro to select the appropriate link.
	for i := range distro.Mirrors {
		if distro.Mirrors[i].Name == entry.Label() {
			link = distro.Mirrors[i].Url
			return link, distro.Mirrors[i].Name, err
		}
	}
	return "", "", fmt.Errorf("Mirror not found: %v", entry.Label())
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

type LogOption struct {
}

func (d *LogOption) Label() string {
	return "Show last log"
}

func getMainMenu(cacheDir string, menus chan<- string) menu.Entry {
	entries := []menu.Entry{}
	if cacheDir != "" {
		// UseCacheOption is a special DirOption represents the root of cache dir
		entries = append(entries, &DirOption{label: "Use Cached ISO", path: cacheDir})
	}
	entries = append(entries, &DownloadOption{})
	entries = append(entries, &LogOption{})

	for {
		// Display the main menu until user makes a valid choice or
		// they encounter an error that's not menu.BackRequest
		entry, err := menu.PromptMenuEntry("Webboot", "Choose an option:", entries, ui.PollEvents(), menus)
		if err != nil && err != menu.BackRequest {
			log.Fatal(err)
		} else if entry != nil {
			return entry
		}
	}
}

func handleError(err error, menus chan<- string) {
	if err == menu.ExitRequest {
		menu.Close()
		os.Exit(0)
	} else if err == menu.BackRequest {
		return
	}

	errorText := err.Error() + "\n" + tmpBuffer.String() + wifiStdout.String() + wifiStderr.String()
	fmt.Fprintln(&logBuffer, errorText)
	menu.DisplayResult(strings.Split(errorText, "\n"), ui.PollEvents(), menus)

	tmpBuffer.Reset()
	wifiStdout.Reset()
	wifiStderr.Reset()
}

func showLog(menus chan<- string) {
	s := logBuffer.String()
	if len(s) > 1024 {
		s = s[len(s)-1024:]
	}
	menu.DisplayResult(strings.Split(s, "\n"), ui.PollEvents(), menus)
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
	verbose("Using cache dir: %v", cacheDir)
	if err := menu.Init(); err != nil {
		log.Fatalf(err.Error())
	}

	menus := make(chan string)
	// Continuously throw away values from menus channel so that the channel doesn't block.
	go func() {
		for {
			<-menus
		}
	}()
	entry := getMainMenu(cacheDir, menus)

	// Buffer the log output, else it might overlap with the menu
	log.SetOutput(&tmpBuffer)

	// check the chosen entry of each level
	// and call it's exec() to get the next level's chosen entry.
	// repeat this process until there is no next level
	var err error

	for entry != nil {
		switch entry.(type) {
		case *LogOption:
			showLog(menus)
			entry = getMainMenu(cacheDir, menus)
		case *DownloadOption:
			// set up network
			progress := menu.NewProgress("Testing network connection", true)
			activeConnection := connected()
			progress.Close()

			if *network && !activeConnection {
				if err := setupNetwork(ui.PollEvents(), menus); err != nil {
					verbose("error on setupNetwork: %+v", err)
				}
			}
			supportedDistros, err = distroData(ui.PollEvents(), menus, cacheDir, "https://raw.githubusercontent.com/u-root/webboot/main/cmds/webboot/distros.json")

			if entry, err = entry.(*DownloadOption).exec(ui.PollEvents(), menus, *network, cacheDir); err != nil {
				handleError(err, menus)
				entry = getMainMenu(cacheDir, menus)
			}
		case *ISO:
			if err = entry.(*ISO).exec(ui.PollEvents(), menus, !*dryRun); err != nil {
				handleError(err, menus)
				entry = getMainMenu(cacheDir, menus)
			}
		case *DirOption:
			dirOption := entry.(*DirOption)
			if entry, err = dirOption.exec(ui.PollEvents(), menus); err != nil {
				// Check if user requested to go back from a cache subdirectory,
				// so we can send them to a DirOption for the parent directory
				if err == menu.BackRequest && dirOption.path != cacheDir {
					entry = &DirOption{path: filepath.Dir(dirOption.path)}
				} else {
					// Otherwise they either requested to go back from the
					// cache root, so we can send them to main menu,
					// or they encountered an error
					handleError(err, menus)
					entry = getMainMenu(cacheDir, menus)
				}
			}
		default:
			handleError(fmt.Errorf("Unknown menu type %T!\n", entry), menus)
			entry = getMainMenu(cacheDir, menus)
		}
	}
}
