// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os/exec"

	"github.com/u-root/webboot/pkg/mountkexec"
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
		{"GoodTest", "testIso", "/tmp/mountDir", ""},
	}

	for _, test := range tests {
		log.Print(test.name)
		if err := mountkexec.MountISO(test.iso, test.dir); err != nil {
			log.Print(err)
		} else {
			o, err := exec.Command("ls", "/tmp/mountDir").CombinedOutput()
			if err != nil {
				log.Fatalf("ls failed: error %v", err)
			}
			log.Printf("%v", string(o))
		}
	}

	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
