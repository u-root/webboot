package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/menu"
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

//	-ifName:  Name of the interface
//	-timeout: Lease timeout in seconds
//	-retry:   Number of DHCP renewals before exiting
//	-verbose: Verbose mode
//	-ipv4:    Use IPV4
//	-ipv6:    Use IPV6
func setUpNetwork(uiEvents <-chan ui.Event) (bool, error) {

	isIfName := func(input string) (string, string, bool) {
		if input[0] == 'e' || input[0] == 'w' {
			return input, "", true
		}
		return "", "not a valid interface name", false
	}

	ifName, err := menu.NewInputWindow("Enter name of the interface:", isIfName, uiEvents)
	if err != nil {
		return false, err
	}

	cl := make(chan string)
	go dhclient.Request(ifName, 15, 5, *v, true, true, cl)
	for {
		msg, ok := <-cl
		if !ok {
			return false, nil
		}
		if msg == "Successful" {
			menu.DisplayResult([]string{msg}, uiEvents)
			return true, nil
		}
		if _, err := menu.DisplayResult([]string{msg}, uiEvents); err != nil {
			return false, err
		}
	}
}
