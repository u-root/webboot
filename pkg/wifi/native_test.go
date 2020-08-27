// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wifi

import (
	"bytes"
	"testing"
)

func TestNative(t *testing.T) {
	// Some things may fail as there may be no wlan or we might not
	// have the right privs. So just bail out of the test if some early
	// ops fail.
	var stdout, stderr bytes.Buffer
	w, err := NewNativeWorker(&stdout, &stderr, "wlan0")
	if err != nil {
		t.Log(err)
		return
	}
	t.Logf("Native is %v", w)
	err = w.Connect(&stdout, &stderr)
	if err != nil {
		t.Log(err)
		return
	}
}
