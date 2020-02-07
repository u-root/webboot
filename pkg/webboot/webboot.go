// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webboot

//Distro defines an operating system distribution
type Distro struct {
	Kernel       string
	Initrd       string
	Cmdline      string
	DownloadLink string
}
