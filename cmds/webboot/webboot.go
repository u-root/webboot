package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/u-root/u-root/pkg/boot/diskboot"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/menus"
)

const (
	tcURL = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
)

//DownloadDistro contains information of downloadable isos.
type DownloadDistro struct {
	URL string
}

// bookmark record the list of iso webboot allow user to download,
// and users can enter these label to indicate an iso to download.
var bookmark = map[string]*DownloadDistro{
	"tinycore": &DownloadDistro{
		tcURL,
	},
}

// kernelPath is for hardcode the path to kernel. should be replaced by u-root later
var kernelPath = map[string]string{
	"archlinux-2020.06.01-x86_64.iso": "arch/boot/x86_64/vmlinuz",
	"TinyCore-11.1.iso":               "boot/vmlinuz",
}

var (
	isoDir  = flag.String("dir", "", "set the iso directory path")
	timeout = flag.Int("timeout", 15, "Lease timeout in seconds")
	retry   = flag.Int("retry", 5, "Max number of attempts for DHCP clients to send requests. -1 means infinity")
	v       = flag.Bool("verbose", false, "Verbose output")
	ipv4    = flag.Bool("ipv4", true, "use IPV4")
	ipv6    = flag.Bool("ipv6", true, "use IPV6")
	devGlob = flag.String("dev", "/sys/class/block/*", "Device glob")

	verbose = func(string, ...interface{}) {}
)

// ISO type contains information of a iso file, incluing its name,
// its filepath, the path to its config files and if it should be choose by default.
type ISO struct {
	name      string
	path      string
	config    []*diskboot.Config
	isDefault bool
}

// DownloadOption contains information of the iso user want to download,
// include it's name and the download URL
type DownloadOption struct {
	isoName string
	url     string
}

// DownloadByLinkOption contains information of the iso user want to download,
// include it's name and the download URL
type DownloadByLinkOption struct {
	isoName string
	url     string
}

// IsDefault is for mark whether this iso is a default choice.
func (u ISO) IsDefault() bool {
	return u.isDefault
}

func (u ISO) String() string {
	return fmt.Sprintf("%+v\n", u)
}

// Label is the string this iso displays in the menu page.
func (u ISO) Label() string {
	return u.name
}

// Do is what will be called after user choose this iso option,
func (u ISO) Do() error {
	if err := getKernel(&u); err != nil {
		return err
	}

	return nil
}

// IsDefault is for mark whether this entry is a default choice.
// We assume download option always be a default option.
func (u DownloadOption) IsDefault() bool {
	return true
}

func (u DownloadOption) String() string {
	return fmt.Sprintf("%+v\n", u)
}

// Label is the string this entry displays in the menu page.
// Download option will be displayed as "Download" in the menu page.
func (u DownloadOption) Label() string {
	return "Download"
}

// Do is what will be called after user choose this download option,
func (u DownloadOption) Do() error {
	// If download option is chosen, user can input the name of iso they want to download.
	// The name of iso should be include in the bookmark.
	isonameCheckFunc := func(input string) (bool, string) {
		if _, ok := bookmark[input]; ok {
			return true, ""
		}
		return false, "not able to find this distro. please try another."
	}
	introwords := "Please input the name of iso you want to download and install here:"
	wid := 100
	ht := 1
	isoLabel, err := menus.NewInputWindow(introwords, wid, ht, isonameCheckFunc)
	if err != nil {
		return err
	}
	u.isoName = isoLabel
	u.url = bookmark[isoLabel].URL

	verbose("arrave download step, URL of iso is %s\n", u.url)

	if err := downloadIso(filepath.Join("/tmp/", u.isoName+".iso"), u.url); err != nil {
		return err
	}
	verbose("iso is downloaded")
	return nil
}

// IsDefault is for mark whether this entry is a default choice.
func (u DownloadByLinkOption) IsDefault() bool {
	return true
}

func (u DownloadByLinkOption) String() string {
	return fmt.Sprintf("%+v\n", u)
}

// Label is the string this entry displays in the menu page.
func (u DownloadByLinkOption) Label() string {
	return "Download by link"
}

// Do is what will be called after user choose this download option,
func (u DownloadByLinkOption) Do() error {
	// if download by link option is chosen, user can input a link to download the iso.
	stringCheckFunc := func(input string) (bool, string) {
		return true, ""
	}
	introwords := "Please input the link of iso you want to download and install here:"
	wid := 100
	ht := 1
	isoLink, err := menus.NewInputWindow(introwords, wid, ht, stringCheckFunc)
	if err != nil {
		return err
	}
	isoName, err := menus.NewInputWindow("Please input the name of iso:", 100, 1, stringCheckFunc)
	if err != nil {
		return err
	}
	u.isoName = isoName
	u.url = isoLink
	verbose("arrave download step, URL of iso is %s\n", u.url)
	if err := downloadIso(filepath.Join("/tmp/", u.isoName+".iso"), u.url); err != nil {
		return err
	}
	verbose("iso is downloaded")

	return nil
}

// getKernel is to find the kernel of a ISO.
func getKernel(u *ISO) error {

	/*
	* TODO: code this function after FindDevice is ready
	 */

	menus.DisplayResult(strings.Fields(fmt.Sprintf("%+v", u)))
	return nil
}

// isoMenu is to  build the iso menu page and return the iso path
// and kernel path of which user choose.
func isoMenu(isos []ISO) error {

	var entries []menus.Entry
	for _, iso := range isos {
		entries = append(entries, iso)
	}
	var downloadOption DownloadOption
	var downloadByLinkOption DownloadByLinkOption
	entries = append(entries, downloadOption, downloadByLinkOption)

	menuTitle := "ISO Menu"
	introwords := "Choose an iso you want to boot (hit enter to choose the default - 1 is the default option) >"
	location := 0
	entry, err := menus.DisplayMenu(menuTitle, introwords, location, entries)

	// if index is exceed the length of entries or less than 0 means something wrong happened
	if err != nil {
		return err
	}
	err = entry.Do()
	return err
	return nil
}

// linkOpen returns an io.ReadCloser that holds the content of the URL
func linkOpen(URL string) (io.ReadCloser, error) {
	switch {
	case strings.HasPrefix(URL, "file://"):
		return os.Open(URL[7:])
	case strings.HasPrefix(URL, "http://"), strings.HasPrefix(URL, "https://"):
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

// downloadIso will download a iso from URL and save it under savePath directory
func downloadIso(isoPath, URL string) error {
	verbose("Should begin to download here..., download url is %s\n", URL)
	introwords := "Please input the name of the interface:"
	wid := 100
	ht := 1
	interfaceNameCheckFunc := func(input string) (bool, string) {
		if input[0] == 'e' || input[0] == 'w' {
			return true, ""
		}
		return false, "not a valid interface name"
	}
	ifName, err := menus.NewInputWindow(introwords, wid, ht, interfaceNameCheckFunc)
	if err != nil {
		return err
	}
	if *ipv4 || *ipv6 {
		dhclient.Request(ifName, *timeout, *retry, *v, *ipv4, *ipv6)
	}
	isoReader, err := linkOpen(URL)
	if err != nil {
		return err
	}
	defer isoReader.Close()
	isofile, err := os.Create(isoPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(isofile, isoReader); err != nil {
		return fmt.Errorf("Error copying to persistent memory device: %v", err)
	}
	if err = isofile.Close(); err != nil {
		return fmt.Errorf("Error closing %s: %v", isoPath, err)
	}

	verbose("downloaded isoPath is %s", isoPath)
	return nil
}

func cleanDevices(devices []*diskboot.Device) {
	for _, device := range devices {
		if err := device.Unmount(mount.MNT_FORCE); err != nil {
			log.Printf("Error unmounting device %s: %v", device, err)
		}
	}
}

func main() {
	flag.Parse()

	if *v {
		verbose = log.Printf
	}

	var isos []ISO

	var devices []*diskboot.Device

	// If isoDir is not given, that means we need use findDevice to find the USB disk
	// else we walk through the given folder to find all iso inside.
	if *isoDir == "" {
		devices = diskboot.FindDevices(*devGlob)
		defer cleanDevices(devices)
		verbose("Got devices: %#v", devices)
		for _, device := range devices {
			for _, iso := range device.Isos {
				t_iso := ISO{
					name:      iso.Name,
					path:      iso.Path,
					config:    iso.Config,
					isDefault: false,
				}
				isos = append(isos, t_iso)
			}
		}

	} else {
		walkfunc := func(path string, info os.FileInfo, err error) error {
			if info.IsDir() == false && filepath.Ext(path) == ".iso" {
				var iso = ISO{info.Name(), path, nil, true}
				isos = append(isos, iso)
			}
			return nil
		}
		filepath.Walk(*isoDir, walkfunc)
	}

	/*err := isoMenu(isos)
		if err != nil {
	        cleanDevices(devices)
			log.Fatalf("%v", err)
		}*/
	return
}
