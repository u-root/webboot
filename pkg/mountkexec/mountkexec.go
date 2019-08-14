// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mountkexec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/kexec"
	"github.com/u-root/u-root/pkg/loop"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/multiboot"
	"github.com/u-root/u-root/pkg/uio"
	"github.com/u-root/webboot/pkg/webboot"
	"golang.org/x/sys/unix"
)

// MountISO creates a directory and mounts an input iso on the directory.
func MountISO(isoPath, mountDir string) error {
	if err := os.MkdirAll(mountDir, 777); err != nil {
		return fmt.Errorf("error making mount directory:%v", err)
	}
	loopDevice, err := loop.FindDevice()
	if err != nil {
		return fmt.Errorf("error finding loop device:%v", err)
	}
	if err := loop.SetFile(loopDevice, isoPath); err != nil {
		return fmt.Errorf("error setting loop device:%v", err)
	}
	var flags uintptr
	flags |= unix.MS_RDONLY
	if err = mount.Mount(loopDevice, mountDir, "iso9660", "", flags); err != nil {
		return fmt.Errorf("error mounting ISO:%v", err)
	}
	return nil
}

// KexecISO boots up a new kernel and initramfs
func KexecISO(opp *webboot.Distro, dir string) error {
	var image boot.OSImage
	kernelPath := opp.Kernel
	if !filepath.IsAbs(kernelPath) {
		kernelPath = filepath.Join(dir, kernelPath)
	}

	if err := multiboot.Probe(kernelPath); err == nil {
		image = &boot.MultibootImage{
			Path:    kernelPath,
			Cmdline: opp.Cmdline,
		}
	} else {
		image = &boot.LinuxImage{
			Kernel:  uio.NewLazyFile(kernelPath),
			Initrd:  uio.NewLazyFile(dir + opp.Initrd),
			Cmdline: opp.Cmdline,
		}
	}
	if err := image.Load(true); err != nil {
		return fmt.Errorf("error failed to kexec into new kernel:%v", err)
	}
	if err := kexec.Reboot(); err != nil {
		return fmt.Errorf("error failed to Reboot into new kernel:%v", err)
	}
	return nil
}
