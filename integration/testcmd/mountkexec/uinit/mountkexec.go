// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os/exec"

	"github.com/u-root/u-root/pkg/cmdline"
	"github.com/u-root/webboot/pkg/mountkexec"
	"github.com/u-root/webboot/pkg/webboot"
	"golang.org/x/sys/unix"
)

func main() {
	var tests = []struct {
		name string
		iso  string
		dir  string
		err  string
	}{
		{"Empty Directory Name", "testIso", "", "error making mount directory:mkdir : no such file or directory"},
		{"Non-existent Iso", "non-existentfile", "/tmp/mountDir", "error setting loop device:open non-existentfile: no such file or directory"},
		{"GoodTest", "testIso", "/tmp/mountDir/", ""},
	}

	for _, test := range tests {
		log.Print(test.name)
		if err := mountkexec.MountISO(test.iso, test.dir); err != nil {
			log.Print(err)
		} else {
			o, err := exec.Command("ls", "/tmp/mountDir/").CombinedOutput()
			if err != nil {
				log.Fatalf("ls failed: error %v", err)
			}
			log.Printf("%v", string(o))
		}
	}

	log.Printf("TestingKernel")
	//TestDistro to be kexeced.
	distro := &webboot.Distro{"kernel", "initiso.cpi", "console=ttyS0", "https:louis.com"}
	kexecCounter, ok := cmdline.Flag("kexeccounter")
	if !ok {
		kexecCounter = "0"
	}
	log.Printf("KEXECCOUNTER=%s\n", kexecCounter)
	if kexecCounter == "0" {
		//Pass kexeccounter to new kernel through commandline.
		distro.Cmdline += " kexeccounter=1"

		if err := mountkexec.KexecISO(distro, "/tmp/mountDir/"); err != nil {
			log.Print(err)
		}
	} else {
		unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	}

}
