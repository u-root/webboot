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

// downloadList record the list of iso webboot allow user to download, and users can enter these label to indicate an iso to download. I know that may be not flexiable enough, but I can not come up with a better way to let webboot know where it can download a distro without this list. Allow user directly input the website to download iso? i don't think that's a good idea too.
var downloadList = map[string]string{"core": "http://tinycorelinux.net/10.x/x86/release/CorePlus-current.iso"}

var (
	isoDir = flag.String("dir", "/", "set the iso directory path")
)

// ISO type contains information of a iso file, incluing its name, its filepath, the path to its kernel and if it should be choose by default.
type ISO struct {
	name      string
	path      string
	kernel    string
	isDefault bool
}

//DownloadOption contains information of the iso user want to download, include it's name and the download URL
type DownloadOption struct {
	isoName string
	url     string
}

// IsDefault is for mark whether this iso is a default choise.
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

// IsDefault is for mark whether this entry is a default choise. we assume download option always be a default option.
func (u DownloadOption) IsDefault() bool {
	return true
}

func (u DownloadOption) String() string {
	return fmt.Sprintf("%+v\n", u)
}

// Label is the string this entry displays in the menu page. download option will be displayed as "Download" in the menu page.
func (u DownloadOption) Label() string {
	return "Download"
}

// getKernel is to find the kernel of a ISO. if the path of ISO is given, just check if the kernel is exist.
func getKernel(u *ISO) error {

	log.Println("try mount iso...")

	isoName := u.name

	diskFile, err := ioutil.TempDir("", "/mnt-iso")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(diskFile)

	mountPath := filepath.Join(diskFile, isoName)
	if mp, err := mount.Mount(u.path, mountPath, "iso9660", "", mount.ReadOnly); err != nil {
		log.Printf("TryMount %s = %v, want nil", u.path, err)
	} else {
		log.Printf("mounted disk file is %s\n", mountPath)
		// if kernel is given, check its validation
		if u.kernel != "" {
			walkfunc := func(path string, info os.FileInfo, err error) error {
				if path == filepath.Join(mountPath, u.kernel) {
					log.Printf("Kernel at %s is found\n", u.kernel)
				}
				return nil
			}

			log.Println("\nfinding kernel...")
			filepath.Walk(diskFile, walkfunc)
			log.Printf("Kernel path is %s\n", u.kernel)
		}

		log.Println("\ntry unmount iso...")
		if err := mp.Unmount(0); err != nil {
			log.Printf("Unmount(%q) = %v, want nil", mountPath, err)
		}
	}

	log.Println("Done")
	return nil
}

// isoMenu is to  build the iso menu page and return the iso path and kernel path of which user choose.
func isoMenu(isos []ISO) (string, string) {

	var entries []menus.Entry
	for _, iso := range isos {
		entries = append(entries, iso)
	}
	var downloadOption DownloadOption
	entries = append(entries, downloadOption)

	index, err := menus.DisplayMenu("ISO Menu", "Choose an iso you want to boot (hit enter to choose the default - 1 is the default option) >", 0, entries)

	// if index is exceed the length of entries or less than 0 means something wrong happened
	if err != nil || index < 0 || index >= len(entries) {
		if err != nil {
			log.Println(err)
		}
		return "", ""
	}
	// if index is the same as the length of entries - 1, it means the chosen option is download. Elsewise the chosen option is a cached iso.
	if index == len(entries)-1 {
		// if download option is chosen, webboot will show another input window for user to indicate what they want
		isoLabel, err := menus.NewInputWindow("Please input the name of iso you want to download and install here:", 100, 1, func(input string) (bool, string) {
			if _, ok := downloadList[input]; ok {
				return true, ""
			}
			return false, "not able to find this distro. please try another."
		})
		if err != nil {
			log.Println(err)
			return "", ""
		}
		downloadOption.isoName = isoLabel
		downloadOption.url = downloadList[isoLabel]
		log.Printf("arrave download step, URL of iso is %s\n", downloadOption.url)
		//  download part need to be test so I temporarily hidden them .
		/*if err := downloadIso(filepath.Join(*isoDir, isoLabel+'.iso'), downloadOption.url); err!=nil{
		      log.Printf("%v", err)
		  }else{
		      log.Printf("iso is downloaded")
		  }*/
	} else {
		var chosenISO = isos[index]
		if err := getKernel(&chosenISO); err != nil {
			log.Println(err)
			return "", ""
		}
		return chosenISO.path, chosenISO.kernel
	}
	return "", ""
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
	timeout := 15
	retry := 5
	ipv4 := true
	ipv6 := true
	verbose := false
	ifName, err := menus.NewInputWindow("Please input the name of the interface:", 100, 1, func(input string) (bool, string) {
		if input[0] == 'e' || input[0] == 'w' {
			return true, ""
		} 
		return false, "not a valid interface name"
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("interface input error")
	}
	dhclient.Request(ifName, timeout, retry, verbose, ipv4, ipv6)
	isoReader, err := linkOpen(URL)
	if err != nil {
		log.Fatal(err)
	}
	defer isoReader.Close()
	isofile, err := os.Create(isoPath)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		if _, err := io.Copy(isofile, isoReader); err != nil {
			log.Fatalf("Error copying to persistent memory device: %v", err)
		}
		if err = isofile.Close(); err != nil {
			log.Fatalf("Error closing %s: %v", isoPath, err)
		}
	}
	return nil
}

func main() {
	flag.Parse()

	// hardcode kernel path
	kernelPath["archlinux-2020.06.01-x86_64.iso"] = "arch/boot/x86_64/vmlinuz"
	kernelPath["TinyCore-11.1.iso"] = "boot/vmlinuz"

	var isos []ISO

	// find all iso inside the given directory
	walkfunc := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() == false && filepath.Ext(path) == ".iso" {
			var iso = ISO{info.Name(), path, kernelPath[info.Name()], true}
			isos = append(isos, iso)
		}
		return nil
	}

	filepath.Walk(*isoDir, walkfunc)

	isoPath, isoKernel := isoMenu(isos)
	log.Printf("isoPath is %s, isoKernel is %s", isoPath, isoKernel)
	return
}
