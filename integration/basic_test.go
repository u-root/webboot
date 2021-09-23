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

var expectString = map[string]string{
	"Arch":       "TODO_PLEASE_SET_EXPECT_STRING",
	"CentOS 7":   "TODO_PLEASE_SET_EXPECT_STRING",
	"CentOS 8":   "TODO_PLEASE_SET_EXPECT_STRING",
	"Debian":     "TODO_PLEASE_SET_EXPECT_STRING",
	"Fedora":     "TODO_PLEASE_SET_EXPECT_STRING",
	"Kali":       "TODO_PLEASE_SET_EXPECT_STRING",
	"Linux Mint": "TODO_PLEASE_SET_EXPECT_STRING",
	"Manjaro":    "TODO_PLEASE_SET_EXPECT_STRING",
	"TinyCore":   "5.4.3-tinycore64",
	"Ubuntu":     "TODO_PLEASE_SET_EXPECT_STRING",
}

func TestScript(t *testing.T) {
	webbootDistro := os.Getenv("WEBBOOT_DISTRO")
	if _, ok := expectString[webbootDistro]; !ok {
		if webbootDistro == "" {
			t.Fatal("WEBBOOT_DISTRO is not set")
		}
		t.Fatalf("Unknown distro: %q", webbootDistro)
	}

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
				"../cmds/cli/ci.json:ci.json",
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

	if err := q.Expect(expectString[webbootDistro]); err != nil {
		t.Fatalf("expected %q, got error: %v", expectString[webbootDistro], err)
	}
}
