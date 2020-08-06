package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	// print how many bytes have been writen to the file
	fmt.Printf("\r%s", strings.Repeat(" ", 40))
	fmt.Printf("\rDownloading... %v bytes complete", wc.Total)
	return n, nil
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
	defer f.Close()
	counter := &WriteCounter{}
	if _, err = io.Copy(f, io.TeeReader(isoReader, counter)); err != nil {
		return fmt.Errorf("Fail to copy iso to a persistent memory device: %v", err)
	}
	fmt.Println("\nDone!")
	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
}

// getCachedDirectory recognizes the usb stick that contains the cached directory from block devices,
// and return it's mount point.
func getCachedDirectory() (*mount.MountPoint, error) {
	blockDevs, err := block.GetBlockDevices()
	if err != nil {
		return nil, fmt.Errorf("No available block devices to boot from")
	}

	mountPoints, err := ioutil.TempDir("", "temp-device-")
	if err != nil {
		return nil, fmt.Errorf("Cannot create tmpdir: %v", err)
	}

	for _, device := range blockDevs {
		mp, err := mount.TryMount(filepath.Join("/dev/", device.Name), filepath.Join(mountPoints, device.Name), "", mount.ReadOnly)
		if err != nil {
			continue
		}
		if _, err = os.Stat(filepath.Join(mp.Path, "Image")); err == nil {
			return mp, nil
		}
	}
	return nil, fmt.Errorf("Do not find the cache directory")
}
