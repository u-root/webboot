// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mountkexec

import (
	"fmt"
	"os"

	"github.com/u-root/u-root/pkg/loop"
	"github.com/u-root/u-root/pkg/mount"
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
