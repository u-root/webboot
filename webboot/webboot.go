// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Synopsis:
//     webboot [OPTIONS...] name of bookmark
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
//	-wifi: [essid [WPA [password]]]
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/mountkexec"
	"github.com/u-root/webboot/pkg/webboot"
)

const (
	wbtcpURL  = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso"
	wbcpURL   = "https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/CorePure64.iso"
	tcURL     = "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	coreURL   = "http://tinycorelinux.net/10.x/x86/release/CorePlus-current.iso"
	ubuURL    = "http://releases.ubuntu.com/18.04/ubuntu-18.04.3-desktop-amd64.iso"
	archURL   = "http://mirror.rackspace.com/archlinux/iso/2020.01.01/archlinux-2020.01.01-x86_64.iso"
	tcCmdLine = "memmap=%dG!%dG earlyprintk=ttyS0,115200 console=ttyS0 console=tty0 root=/dev/pmem0 loglevel=3 cde"
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
	wifi     = flag.String("wifi", "", "[essid [WPA [password]]]")
	pmbase   = flag.Int("pmbase", 1, "PM base in GiB")
	pmsize   = flag.Int("pmsize", 1, "PM size in GiB")
	bookmark = map[string]*webboot.Distro{
		// TODO: Fix webboot to process the tinycore's kernel and initrd to boot from instead of using our customized kernel
		"webboot-tinycorepure": &webboot.Distro{
			"boot/vmlinuz64",
			"/boot/corepure64.gz",
			tinyCoreCmdLine(),
			wbtcpURL,
		},
		"webboot-corepure": &webboot.Distro{
			"boot/vmlinuz64",
			"/boot/corepure64.gz",
			tinyCoreCmdLine(),
			wbcpURL,
		},
		"tinycore": &webboot.Distro{
			"boot/vmlinuz64",
			"/boot/corepure64.gz",
			tinyCoreCmdLine(),
			tcURL,
		},
		"Tinycore": &webboot.Distro{
			"/bzImage", // our own custom kernel, which has to be in the initramfs
			"/boot/corepure64.gz",
			tinyCoreCmdLine(),
			tcURL,
		},
		"arch": &webboot.Distro{
			"arch/boot/x86_64/vmlinuz",
			"/arch/boot/x86_64/archiso.img",
			"memmap=1G!1G earlyprintk=ttyS0,115200 console=ttyS0 console=tty0 root=/dev/pmem0 loglevel=3",
			archURL,
		},
		"Arch": &webboot.Distro{
			"/bzImage", // our own custom kernel, which has to be in the initramfs
			"/arch/boot/x86_64/archiso.img",
			"memmap=1G!1G earlyprintk=ttyS0,115200 console=ttyS0 console=tty0 root=/dev/pmem0 loglevel=3",
			archURL,
		},
		"ubuntu": &webboot.Distro{
			"casper/vmlinuz",
			"/casper/initrd",
			"memmap=1G!1G earlyprintk=ttyS0,115200 console=ttyS0 console=tty0 root=/dev/pmem0 loglevel=3 boot=casper file=/cdrom/preseed/ubuntu.seed",
			ubuURL,
		},
		"Ubuntu": &webboot.Distro{
			"/bzImage", // our own custom kernel, which has to be in the initramfs
			"/casper/initrd",
			"memmap=1G!1G earlyprintk=ttyS0,115200 console=ttyS0 console=tty0 root=/dev/pmem0 loglevel=3 boot=casper file=/cdrom/preseed/ubuntu.seed",
			ubuURL,
		},
		"local": &webboot.Distro{
			"/bzImage",
			"/boot/corepure64.gz",
			tinyCoreCmdLine(),
			"file:///iso", // NOTE: three / is REQUIRED
		},
		"core": &webboot.Distro{
			"boot/vmlinuz",
			"/boot/core.gz",
			tinyCoreCmdLine(),
			coreURL,
		},
	}
)

func tinyCoreCmdLine() string {
	return fmt.Sprintf(tcCmdLine, *pmsize, *pmbase)
}

// parseArg takes a name of bookmark and produces a download link
// The download link can be used to download data to a persistent memory device '/dev/pmem0'
func parseArg(arg string) (string, string, error) {
	if u, ok := bookmark[arg]; ok {
		return u.DownloadLink, arg, nil
	}
	return "", "", fmt.Errorf("%s is not supported", arg)
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

// setupWIFI enables connection to a specified wifi network
// wifi can be an open or closed network
func setupWIFI(wifi string) error {
	if wifi == "" {
		return nil
	}

	c := exec.Command("wifi", strings.Split(wifi, " ")...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	// wifi and its children can run a long time. The bigger problem is
	// knowing when the net is ready, but one problem at a time.
	if err := c.Start(); err != nil {
		return fmt.Errorf("Error starting wifi(%v):%v", wifi, err)
	}
	return nil
}

func usage() {
	log.Printf("Usage: %s [flags] URL or name of bookmark\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()

	commandline, err := ioutil.ReadFile("/proc/cmdline")


	if err != nil {
		fmt.Errorf("Unable to read /proc/cmdline: %v", err)
		//maybe here, we can have a hardcoded commandline so if
		//we cannot get /proc/cmdline, we can just use the hardcoded on
		//just an idea. 
	}

	_ = commandline //do something with the output of /proc/cmdline


	if flag.NArg() != 1 {
		usage()
	}

	URL, filename, err := parseArg(flag.Arg(0))
	if err != nil {
		var s string
		for os := range bookmark {
			s += os + " "
		}
		log.Fatalf("%v, valid names: %q", err, s)
	}

	// TODO: Find a persistent memory device large enough to store the content. If no blocks are available, error the user.
	pmem, err := os.OpenFile("/dev/pmem0", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}

	if *wifi != "" {
		if err := setupWIFI(*wifi); err != nil {
			log.Fatal(err)
		}
	}

	if *ipv4 || *ipv6 {
		dhclient.Request(*ifName, *timeout, *retry, *verbose, *ipv4, *ipv6)
	}

	// Processes the URL to receive an io.ReadCloser, which holds the content of the downloaded file
	log.Println("Retrieving the file...")
	iso, err := linkOpen(URL)
	if err != nil {
		log.Fatal(err)
	}
	defer iso.Close()

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
