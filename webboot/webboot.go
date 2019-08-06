// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Get the time the machine has been up
// Synopsis:
//     webboot
package main

import (
	"flag"
	"log"

	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/mountkexec"
	"github.com/u-root/webboot/pkg/webboot"
)

var (
	cmd       = flag.String("cmd", "", "Command Line")
	mountDir  = flag.String("dir", "/tmp/mountDir", "The mount point of the ISO")
	ifName    = flag.String("interface", "^e.*", "Name of the interface")
	timeout   = flag.Int("timeout", 15, "Lease timeout in seconds")
	retry     = flag.Int("retry", 5, "Max number of attempts for DHCP clients to send requests. -1 means infinity")
	verbose   = flag.Bool("verbose", false, "Verbose output")
	ipv4      = flag.Bool("ipv4", true, "use IPV4")
	ipv6      = flag.Bool("ipv6", true, "use IPV6")
	osSystems = map[string]*webboot.Distro{
		"tinycore": &webboot.Distro{"/boot/vmlinuz64", "/boot/corepure64.gz", "console=tty1", "https:louis.com"},
		"TestISO":  &webboot.Distro{"/kernel", "/initiso.cpi", "console=ttyS0 console=tty0", "https:louis.com"},
	}
)

func main() {

	flag.Parse()

	dhclient.Request(*ifName, *timeout, *retry, *verbose, *ipv4, *ipv6)

	if len(flag.Args()) < 1 {
		log.Fatal("error:pass in an operating system")
	}

	osname := flag.Args()[0]

	if _, ok := osSystems[osname]; !ok {
		log.Printf("error operating system is not supported,The following systems are supported")
		for os := range osSystems {
			log.Printf("%v", os)
		}
		log.Fatalf("exit")
	}
	//Mount the downloaded ISO
	if err := mountkexec.MountISO(osname, *mountDir); err != nil {
		log.Fatalf("error in mountISO:%v", err)
	}
	//Process the commandline input of the distro
	if cmdline, err := webboot.CommandLine(osSystems[osname].Cmdline, *cmd); err != nil {
		log.Fatalf("error in webbootCommandline:%v", err)
	} else {
		osSystems[osname].Cmdline = cmdline
	}

	//Kexec into the new distro
	if err := mountkexec.KexecISO(osSystems[osname], *mountDir); err != nil {
		log.Fatalf("error in kexecISO:%v", err)
	}
}
