package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/menus"
)

// kernelPath is for hardcode the path to kernel. should be replaced by u-root later
var kernelPath = make(map[string]string)


const (
	tcURL     = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
)

type Distro struct {
	URL     string
}

// bookmark record the list of iso webboot allow user to download,
// and users can enter these label to indicate an iso to download.
var bookmark = map[string]*Distro{
	"tinycore": &Distro{
		tcURL,
	}
}

var (
	isoDir  = flag.String("dir", "", "set the iso directory path")
	timeout = flag.Int("timeout", 15, "Lease timeout in seconds")
	retry   = flag.Int("retry", 5, "Max number of attempts for DHCP clients to send requests. -1 means infinity")
	verbose = flag.Bool("verbose", false, "Verbose output")
	ipv4    = flag.Bool("ipv4", true, "use IPV4")
	ipv6    = flag.Bool("ipv6", true, "use IPV6")
)

// ISO type contains information of a iso file, incluing its name,
// its filepath, the path to its kernel and if it should be choose by default.
type ISO struct {
	name      string
	path      string
	kernel    string
	isDefault bool
}

// DownloadOption contains information of the iso user want to download,
// include it's name and the download URL
type DownloadOption struct {
	isoName string
	url     string
}

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
func (u ISO) Do() error{
	if err := getKernel(&u); err != nil {
		return err
	}
	if(*verbose){
		log.Printf("isoPath is %s, isoKernel is %s", chosenISO.path, chosenISO.kernel)
	}
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
func (u DownloadOption) Do() error{
	// If download option is chosen, user can input the name of iso they want to download.
	// The name of iso should be include in the bookmark.
	isoname_check_func = func(input string) (bool, string) {
		if _, ok := downloadList[input]; ok {
			return true, ""
		}
		return false, "not able to find this distro. please try another."
	}
	isoLabel, err := menus.NewInputWindow("Please input the name of iso you want to download and install here:", 100, 1, isoname_check_func)
	if err != nil {
		log.Println(err)
		return "", ""
	}
	downloadOption.isoName = isoLabel
	downloadOption.url = downloadList[isoLabel].URL

	if(*verbose){
		log.Printf("arrave download step, URL of iso is %s\n", downloadOption.url)
	}
	if err := downloadIso(filepath.Join(*isoDir, isoLabel+".iso"), downloadOption.url); err!=nil{
		return err
	}else if(*verbose){
		log.Printf("iso is downloaded")
	}
}

func (u DownloadByLinkOption) IsDefault() bool {
	return true
}

func (u DownloadODownloadByLinkOptionption) String() string {
	return fmt.Sprintf("%+v\n", u)
}

func (u DownloadByLinkOption) Label() string {
	return "Download by link"
}

func (u DownloadOption) Do() error{
	// if download by link option is chosen, user can input a link to download the iso. 
	string_check_func = func(input string) (bool, string) {
		return true, ""
	}
	isoLink, err := menus.NewInputWindow("Please input the link of iso you want to download and install here:", 100, 1, string_check_func)
	if err != nil {
		return err
	}
	isoName, err := menus.GetInput("Please input the name of iso:", 4, 100, 1, string_check_func)
	if err != nil {
		return err
	}
	downloadByLinkOption.isoName = isoName
	downloadByLinkOption.url = isoLink
	if(*verbose){
		log.Printf("arrave download step, URL of iso is %s\n", downloadByLinkOption.url)
	}
	if err := downloadIso(filepath.Join(*isoDir, isoName+".iso"), downloadByLinkOption.url); err!=nil{
		return err
	}else if(*verbose){
		log.Printf("iso is downloaded")
	}

}

// getKernel is to find the kernel of a ISO. if the path of ISO is given,
// just check if the kernel is exist.
func getKernel(u *ISO) error {

	if(*verbose){
		log.Println("try mount iso...")
	}

	isoName := u.name

	diskFile, err := ioutil.TempDir("", "/mnt-iso")
	if err != nil {
		return   err
	}
	defer os.RemoveAll(diskFile)

	mountPath := filepath.Join(diskFile, isoName)
	if mp, err := mount.Mount(u.path, mountPath, "iso9660", "", mount.ReadOnly); err != nil {
		return fmt.Errorf(("TryMount %s = %v, want nil", u.path, err)
	} else {
		if(*verbose){
			log.Printf("mounted disk file is %s\n", mountPath)
		}
		// if kernel is given, check its validation
		if u.kernel != "" {
			walkfunc := func(path string, info os.FileInfo, err error) error {
				if path == filepath.Join(mountPath, u.kernel) {
					log.Printf("Kernel at %s is found\n", u.kernel)
				}
				return nil
			}

			if(*verbose){
				log.Println("\nfinding kernel...")
			}
			filepath.Walk(diskFile, walkfunc)
			if(*verbose){
				log.Printf("Kernel path is %s\n", u.kernel)
			}
		}

		log.Println("\ntry unmount iso...")
		if err := mp.Unmount(0); err != nil {
			return fmt.Errorf("Unmount(%q) = %v, want nil", mountPath, err)
		}
	}

	if(*verbose){
		log.Println("Done")
	}
	return nil
}

// isoMenu is to  build the iso menu page and return the iso path
// and kernel path of which user choose.
func isoMenu(isos []ISO) (error) {

	var entries []menus.Entry
	for _, iso := range isos {
		entries = append(entries, iso)
	}
	var downloadOption DownloadOption{"Download"}
	var downloadByLinkOption DownloadOption{"Download By Link"}
	entries = append(entries, downloadOption, downloadByLinkOption)

	index, err := menus.DisplayMenu("ISO Menu", "Choose an iso you want to boot (hit enter to choose the default - 1 is the default option) >", 0, entries)

	// if index is exceed the length of entries or less than 0 means something wrong happened
	if err != nil {
		return err
	}else if index < 0 || index >= len(entries) {
		return fmt.Errorf("input index is less than 0 or exceed the length of option list")
	}
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
	log.Printf("Should begin to download here..., download url is %s\n", URL)
	ifName, err := menus.NewInputWindow("Please input the name of the interface:", 100, 1, func(input string) (bool, string) {
		if input[0] == 'e' || input[0] == 'w' {
			return true, ""
		}
		return false, "not a valid interface name"
	})
	if err != nil {
		return err
	}
	if *ipv4 || *ipv6 {
		dhclient.Request(ifName, *timeout, *retry, *verbose, *ipv4, *ipv6)
	}
	isoReader, err := linkOpen(URL)
	if err != nil {
		return err
	}
	defer isoReader.Close()
	isofile, err := os.Create(isoPath)
	if err != nil {
		return err
	} else {
		if _, err := io.Copy(isofile, isoReader); err != nil {
			return fmt.Errorf("Error copying to persistent memory device: %v", err)
		}
		if err = isofile.Close(); err != nil {
			return fmt.Errorf("Error closing %s: %v", isoPath, err)
		}
	}

	if(*verbose){
		log.Printf("downloaded isoPath is %s", isoPath)
	}
	return nil
}

func main() {
	flag.Parse()

	// hardcode kernel path
	kernelPath["archlinux-2020.06.01-x86_64.iso"] = "arch/boot/x86_64/vmlinuz"
	kernelPath["TinyCore-11.1.iso"] = "boot/vmlinuz"

	var isos []ISO

	// If dir is null, that means we need use findDevice to find the USB disk
	// else we walk through the given folder to find all iso inside.
	if(*dir == ""){

	} else {
		walkfunc := func(path string, info os.FileInfo, err error) error {
			if info.IsDir() == false && filepath.Ext(path) == ".iso" {
				var iso = ISO{info.Name(), path, kernelPath[info.Name()], true}
				isos = append(isos, iso)
			}
			return nil
		}
		filepath.Walk(*isoDir, walkfunc)
	}

	err := isoMenu(isos)
	if(err != nil){
		log.Fatalf(err)
	}
	return
}
