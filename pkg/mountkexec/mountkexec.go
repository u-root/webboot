// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mountkexec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/boot/kexec"
	"github.com/u-root/u-root/pkg/boot/multiboot"
	"github.com/u-root/u-root/pkg/uio"
	"github.com/u-root/webboot/pkg/webboot"
)

// KexecISO boots up a new kernel and initramfs
func KexecISO(opp *webboot.Distro, dir string) error {
	var image boot.OSImage
	kernelPath := opp.Kernel

	if !filepath.IsAbs(kernelPath) {
		kernelPath = filepath.Join(dir, kernelPath)
	}

	//kernel can be used in multiboot.Probe
	kernel, err := os.Open(kernelPath)

	if err != nil {
		fmt.Errorf("KexecISO error opening kernelPath %v: ", err)
	}

	if err := multiboot.Probe(kernel); err == nil {
		image = &boot.MultibootImage{
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
