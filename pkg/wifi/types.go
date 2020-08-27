// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wifi

import "io"

type Option struct {
	Essid     string
	AuthSuite SecProto
}

type WiFi interface {
	Scan(stdout, stderr io.Writer) ([]Option, error)
	GetID(stdout, stderr io.Writer) (string, error)
	Connect(stdout, stderr io.Writer, a ...string) error
}
