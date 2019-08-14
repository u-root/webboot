// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Synopsis:
//     webboot [OPTIONS...] URL or name of bookmark
//
// Options:
//	-cmd: Command line parameters to the second kernel
//	-mountDir: Mount point
//	-ifName: Name of the interface
//	-timeout: Lease timeout in seconds
//	-retry: Number of DHCP renewals before exiting
//	-verbose:  Verbose output
//	-ipv4: Use IPV4
//	-ipv6: Use IPV6
//	-dryrun: Do not do the kexec
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
	bookmark = map[string]*webboot.Distro{
		"tinycore": &webboot.Distro{"boot/vmlinuz64", "/boot/corepure64.gz", "console=tty0", "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"},
		"mykernel": &webboot.Distro{"/bzImage", "/boot/corepure64.gz", "console=tty0", "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"},
		"core":     &webboot.Distro{"boot/vmlinuz", "/boot/core.gz", "console=tty0", "http://tinycorelinux.net/10.x/x86/release/CorePlus-current.iso"},
	}
)

// parseArg takes a name and produces a filename and a URL
// The URL can be used to download data to the file 'filename'
// The argument is either a full URL or a bookmark.
func parseArg(arg string) (string, string, error) {
	var URL, filename string
	if u, ok := bookmark[arg]; ok {
		return u.DownloadLink, arg, nil
	}

	var err error
	filename, err = name(arg)
	URL = arg
	if err != nil {
		return "", "", fmt.Errorf("%v is not a valid URL: %v", URL, err)
	}

	return URL, filename, nil
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

// write copies the content from an io.readCloser to a named file with filename.
func write(read io.ReadCloser, filename string) error {
	w, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer w.Close()
	defer read.Close()

	if _, err := io.Copy(w, read); err != nil {
		return fmt.Errorf("Error copying %v: %v", filename, err)
	}
	return nil
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

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
	}

	dhclient.Request(*ifName, *timeout, *retry, *verbose, *ipv4, *ipv6)

	arg := flag.Arg(0)

	URL, filename, err := parseArg(arg)
	if err != nil {
		log.Fatal(err)
	}

	//Processes the URL to receive an io.ReadCloser, which holds the content of the downloaded file
	log.Println("Retrieving the file...")
	read, err := linkOpen(URL)
	if err != nil {
		log.Fatal(err)
	}

	err = write(read, filename)
	if err != nil {
		log.Fatal(err)
	}

	tmp, err := ioutil.TempDir("", "mnt")
	if err != nil {
		log.Fatal(err)
	}

	if err = mountkexec.MountISO(filename, tmp); err != nil {
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
