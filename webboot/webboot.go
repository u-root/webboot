// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Synopsis:
//     webboot [OPTIONS...] URL or name of bookmark
//
// Options:
//	-cmd: Command line parameters to the second kernel
//	-ifName: Name of the interface
//	-timeout: Lease timeout in seconds
//	-retry: Number of DHCP renewals before exiting
//	-verbose:  Verbose output
//	-ipv4: Use IPV4
//	-ipv6: Use IPV6
//	-dryrun: Do not do the kexec
//	-essid: ESSID name
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/mountkexec"
	"github.com/u-root/webboot/pkg/webboot"
)

var (
	cmd      = flag.String("cmd", "", "Command line parameters to the second kernel")
	ifName   = flag.String("interface", "^[we].*", "Name of the interface")
	timeout  = flag.Int("timeout", 15, "Lease timeout in seconds")
	retry    = flag.Int("retry", 5, "Max number of attempts for DHCP clients to send requests. -1 means infinity")
	verbose  = flag.Bool("verbose", false, "Verbose output")
	ipv4     = flag.Bool("ipv4", true, "use IPV4")
	ipv6     = flag.Bool("ipv6", true, "use IPV6")
	dryrun   = flag.Bool("dryrun", false, "Do not do the kexec")
	wifi    = flag.String("wifi", "GoogleGuest", "[essid [WPA [password]]]")
	bookmark = map[string]*webboot.Distro{
//		TODO: Fix webboot to process the tinycore's kernel and initrd to boot from instead of using our customized kernel
//		"tinycore": &webboot.Distro{"boot/vmlinuz64", "/boot/corepure64.gz", "console=tty0", "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"},
		"Tinycore": &webboot.Distro{"/bzImage", "/boot/corepure64.gz", "memmap=4G!4G console=tty1 root=/dev/pmem0 loglevel=3 cde waitusb=5 vga=791", "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"},
//		TODO: Fix 'core' with CorePlus' 64-bit architecture
//		"core":     &webboot.Distro{"boot/vmlinuz", "/boot/core.gz", "console=tty0", "http://tinycorelinux.net/10.x/x86/release/CorePlus-current.iso"},
	}
)

// parseArg takes a name and produces a filename and a URL
// The URL can be used to download data to the file 'filename'
// The argument is either a full URL or a bookmark.
func parseArg(arg string) (string, string, error) {
	if u, ok := bookmark[arg]; ok {
		return u.DownloadLink, arg, nil
	}
	filename, err := name(arg)
	if err != nil {
		return "", "", fmt.Errorf("%v is not a valid URL: %v", arg, err)
	}
	return arg, filename, nil
}

// linkOpen returns an io.ReadCloser that holds the content of the URL
func linkOpen(URL string) (io.ReadCloser, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Get failed: %v", resp.StatusCode)
	}
	return resp.Body, nil
}

// name takes a URL and generates a filename from it
// For example, if a valid URL = http://tinycorelinux.net/10.x/x86_64/release/CorePure64-10.1.iso, then filename = CorePure64-10.1.iso
// if the URL is empty or if the URL's Path ends in /, name returns a default index.html as the filename
func name(URL string) (string, error) {
	p, err := url.Parse(URL)

	if err != nil {
		return "", err
	}
	filename := "index.html"

	if p.Path != "" && !strings.HasSuffix(p.Path, "/") {
		filename = path.Base(p.Path)
	}
	return filename, nil
}

func usage() {
	log.Printf("Usage: %s [flags] URL or name of bookmark\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func setupWIFI(wifi string) error {
	if wifi == "" {
		return nil
	}

	c := exec.Command("wifi", strings.Split(wifi, " ")...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	// wifi and its children can run a long time. The bigger problem is
	// knowing when the net is ready, but one problem at a time.
	if err := c.Start(); err != nil {
		return fmt.Errorf("error starting wifi(%v):%v", wifi, err)
	}
	return nil
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
	}
	if err := setupWIFI(*wifi); err != nil {
		log.Fatal(err)
	}

	if *ipv4 || *ipv6 {
		dhclient.Request(*ifName, *timeout, *retry, *verbose, *ipv4, *ipv6)
	}

	arg := flag.Arg(0)

	URL, filename, err := parseArg(arg)
	if err != nil {
		log.Fatal(err)
	}

	// Processes the URL to receive an io.ReadCloser, which holds the content of the downloaded file
	log.Println("Retrieving the file...")
	iso, err := linkOpen(URL)
	if err != nil {
		log.Fatal(err)
	}
	defer iso.Close()

	// TODO: Find a persistent memory device large enough to store the content. If no blocks are available, error the user.
	pmem, err := os.OpenFile("/dev/pmem0", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(pmem, iso); err != nil {
		log.Fatalf("Error copying to persistent memory device: %v", err)
	}
	if err = pmem.Close(); err != nil {
		log.Fatalf("Error closing /dev/pmem0: %v", err)
	}

	tmp, err := ioutil.TempDir("", "mnt")
	if err != nil {
		log.Fatal(err)
	}

	if err = mountkexec.MountISOPmem("/dev/pmem0", tmp); err != nil {
		log.Fatalf("Error in mountISO:%v", err)

	}
	if *dryrun == false {
		if cmdline, err := webboot.CommandLine(bookmark[filename].Cmdline, *cmd); err != nil {
			log.Fatalf("Error in webbootCommandline:%v", err)
		} else {
			bookmark[filename].Cmdline = cmdline
		}
		if err := mountkexec.KexecISO(bookmark[filename], tmp); err != nil {
			log.Fatalf("Error in kexecISO:%v", err)
		}
	}

	fmt.Printf("The URL requested: %v\n The file requested: %v\n The mounting point: %v\n", URL, filename, tmp)
}
