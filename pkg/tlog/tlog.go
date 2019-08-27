// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tlog

import (
	"testing"
)

//Testing implements testing interface
type Testing struct {
	Test *testing.T
}

//Print prints a input string
func (t Testing) Print(v ...interface{}) {
	t.Test.Log(v...)
}

//Printf prints a formated string
func (t Testing) Printf(format string, v ...interface{}) {
	t.Test.Logf(format, v...)
}
