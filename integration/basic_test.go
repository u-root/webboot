// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race
// +build !race

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/u-root/u-root/pkg/qemu"
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
	// The vmtest packages do not work any more and I'm a bit tired
	// of trying to figure out why. Damn modules.

	if _, err := os.Stat("u-root"); err != nil {
		c := exec.Command("git", "clone", "--single-branch", "https://github.com/u-root/u-root")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			t.Fatalf("cloning u-root: %v", err)
		}
		c = exec.Command("go", "build", ".")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		c.Dir = "u-root"
		if err := c.Run(); err != nil {
			t.Fatalf("cloning u-root: %v", err)
		}
	}

	var fail bool

	var k string

	webbootDistro := os.Getenv("WEBBOOT_DISTRO")
	if _, ok := expectString[webbootDistro]; !ok {
		fail = true
		if webbootDistro == "" {
			t.Errorf("WEBBOOT_DISTRO is not set")
		}
		t.Errorf("Unknown distro: %q", webbootDistro)
	}
	if _, ok := os.LookupEnv("UROOT_INITRAMFS"); !ok {
		fail = true
		t.Errorf("UROOT_INITRAMFS needs to be set")
	}
	if fail {
		t.Fatalf("can not continue due to errors")
	}

	c := exec.Command("./u-root/u-root",
		"-files", "../cmds/cli/ci.json:ci.json",
		"-files", k,
		"-files", "/etc/ssl/certs",
		"../cmds/webboot",
		"../cmds/cli",

		"./u-root/cmds/core/init",
		"./u-root/cmds/core/ip",
		"./u-root/cmds/core/shutdown",
		"./u-root/cmds/core/sleep",
		"./u-root/cmds/core/dhclient",
		"./u-root/cmds/core/elvish",
		"./u-root/cmds/boot/pxeboot")
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	c.Env = append(os.Environ(), "GOARCH=amd64", "GOOS=linux")
	t.Logf("Args %v cmd %v", c.Args, c)
	if err := c.Run(); err != nil {
		t.Fatalf("Running u-root: %v", err)
	}
	q, cleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name: "ShellScript",
		/* it would be so nice if this actually worked.
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
				"/etc/ssl/certs",
			},
		},
		*/
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
			// The webbootDistro may contain spaces.
			fmt.Sprintf("cli -distroName=%q", webbootDistro),
			"shutdown -h",
		},
	})
	defer cleanup()

	if err := q.Expect(expectString[webbootDistro]); err != nil {
		t.Fatalf("expected %q, got error: %v", expectString[webbootDistro], err)
	}
}
