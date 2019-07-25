// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/u-root/webboot/pkg/tlog"

	"github.com/u-root/u-root/pkg/cp"
	"github.com/u-root/u-root/pkg/golang"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/uroot/builder"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
)

// genISO generates an iso containing a kernel and initramfs to be used for testing mount and kexec
func genISO(t *testing.T) (err error) {
	// Copy kernel to tmpDir for tests involving kexec.
	tmpDir, err := ioutil.TempDir("", "uroot-integration")
	if err != nil {
		return err
	}
	kernel := filepath.Join(tmpDir, "kernel")
	if err := cp.Copy(os.Getenv("UROOT_KERNEL"), kernel); err != nil {
		return err
	}

	// OutputFile
	tlogger := &tlog.Testing{t}
	newOutputFile := filepath.Join(tmpDir, "initiso.cpio")
	w, err := initramfs.CPIO.OpenWriter(tlogger, newOutputFile, "", "")
	if err != nil {
		return err
	}

	cmds := []string{"github.com/u-root/u-root/cmds/*"}
	env := golang.Default()
	env.CgoEnabled = false
	env.GOARCH = TestArch()

	// Build u-root
	opts := uroot.Opts{
		Env: env,
		Commands: []uroot.Commands{
			{
				Builder:  builder.BusyBox,
				Packages: cmds,
			},
		},
		TempDir:      tmpDir,
		BaseArchive:  uroot.DefaultRamfs.Reader(),
		OutputFile:   w,
		DefaultShell: "elvish",
	}

	if err := uroot.CreateInitramfs(tlogger, opts); err != nil {
		return err
	}
	o, err := exec.Command("genisoimage", "-o", "/tmp/tempdata.iso", tmpDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %v", o, err)
	}

	return nil
}

// TestMountKexec runs an init which mounts a filesystem and kexecs a kernel.
func TestKexecMount(t *testing.T) {
	if err := genISO(t); err != nil {
		t.Error(err)
	}
	if TestArch() != "amd64" {
		t.Skipf("test not supported on %s", TestArch())
	}

	// Create the CPIO and start QEMU.
	q, cleanup := QEMUTest(t, &Options{
		Cmds: []string{
			"github.com/u-root/webboot/integration/testcmd/mountkexec/uinit",
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-root/cmds/core/ls",
		}, Files: []string{
			"/tmp/tempdata.iso:testIso",
		},
	})
	defer cleanup()
	// Loop through the expected results and compare with qemu's output
	var results = []struct {
		expect string
	}{
		{"Empty Directory Name"},
		{"error making mount directory:mkdir : no such file or directory"},
		{"Non-existent Iso"},
		{"error setting loop device:open non-existentfile: no such file or directory"},
		{"GoodTest"},
		{"kernel"},
	}

	for _, results := range results {
		if err := q.Expect(results.expect); err != nil {
			t.Error(err)
		}
	}

}
