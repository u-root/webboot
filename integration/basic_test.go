// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !race

package integration

import (
	"os"
	"testing"
	"time"

	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/vmtest"
)

func TestScript(t *testing.T) {
	webbootDistro := os.Getenv("WEBBOOT_DISTRO")
	q, cleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name: "ShellScript",
		BuildOpts: uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/init",
				"github.com/u-root/u-root/cmds/core/ip",
				"github.com/u-root/u-root/cmds/core/shutdown",
				"github.com/u-root/u-root/cmds/core/sleep",
				"github.com/u-root/u-root/cmds/boot/pxeboot",
				"github.com/u-root/webboot/cmds/webboot",
				"github.com/u-root/webboot/cmds/cli",
				"github.com/u-root/u-root/cmds/core/dhclient",
				"github.com/u-root/u-root/cmds/core/elvish",
			),
			ExtraFiles: []string{
				"../cmds/cli/ci.json:/ci.json",
				"/sbin/kexec",
			},
		},
		QEMUOpts: qemu.Options{
			Timeout: 300 * time.Second,
			Devices: []qemu.Device{
				qemu.ArbitraryArgs{
					"-machine", "q35",
					"-device", "rtl8139,netdev=u1",
					"-netdev", "user,id=u1",
					"-m", "4G",
				},
			},
			KernelArgs: "UROOT_NOHWRNG=1",
		},
		TestCmds: []string{
			"dhclient -ipv6=f -v eth0",
			"cli -distroName=" + webbootDistro,
			"shutdown -h",
		},
	})
	defer cleanup()

	if err := q.Expect("5.4.3-tinycore64"); err != nil {
		t.Fatal(`expected "5.4.3-tinycore64", got error: `, err)
	}
}
