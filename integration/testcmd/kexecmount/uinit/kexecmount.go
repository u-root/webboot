// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"log"

	"github.com/u-root/u-root/pkg/cmdline"
	"golang.org/x/sys/unix"
)

func main() {
	// Get counter,increment it and return it to mountkexec_test.go
	kExecCounter, ok := cmdline.Flag("kexeccounter")
	if !ok {
		kExecCounter = "0"
	}
	log.Printf("KEXECCOUNTER=%s\n", kExecCounter)
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
