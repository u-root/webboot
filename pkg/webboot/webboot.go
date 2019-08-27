// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webboot

import (
	"github.com/u-root/u-root/pkg/cmdline"
)

//Distro defines an operating system distribution
type Distro struct {
	Kernel       string
	Initrd       string
	Cmdline      string
	DownloadLink string
}

//CommandLine processes the command line arguments of the distro
func CommandLine(distroCmd, commandln string) (string, error) {
	if commandln != "" {
		distroCmd = commandln
	}
	if distroCmd == "reuse-cmdline" {
		procCmdLine := cmdline.NewCmdLine()
		if procCmdLine.Err != nil {
			return "", procCmdLine.Err
		}
		distroCmd = procCmdLine.Raw
	}
	return distroCmd, nil
}
