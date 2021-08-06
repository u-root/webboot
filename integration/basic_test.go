// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !race

package integration

import (
	"testing"
	"time"

	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/vmtest"
)

func TestScript(t *testing.T) {
	webbootDistro := os.GetEnv("WEBBOOT_DISTRO")
	q, cleanup := vmtest.QEMUTest(t, &vmtest.Options{
		Name: "ShellScript",
		BuildOpts: uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/init",
				"github.com/u-root/u-root/cmds/core/elvish",
				"github.com/u-root/u-root/cmds/core/ip",
				"github.com/u-root/u-root/cmds/core/shutdown",
				"github.com/u-root/u-root/cmds/core/sleep",
				"github.com/u-root/u-root/cmds/boot/pxeboot",
				"github.com/u-root/webboot/cmds/webboot",
				"github.com/u-root/webboot/cmds/cli",
				"github.com/u-root/u-root/cmds/core/dhclient",
			),
			ExtraFiles: []string{
				"../cmds/cli/ci.json:/ci.json",
			},
		},
		QEMUOpts: qemu.Options{Kernel: "../linux/arch/x86/boot/bzImage", Timeout: 120 * time.Second},
		TestCmds: []string{
			"dhclient -v",
			"cli -distroName=" + webbootDistro,
			"shutdown -h",
		},
	})
	defer cleanup()

	if err := q.Expect("www.tinycorelinux.net"); err != nil {
		t.Fatal(`expected "www.tinycorelinux.net", got error: `, err)
	}
}
