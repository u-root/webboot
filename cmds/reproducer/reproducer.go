// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wifi

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// IWLWorker implements the WiFi interface using the Intel Wireless LAN commands
type IWLWorker struct {
	Interface string
}

func (w *IWLWorker) main(stdout, stderr io.Writer, a ...string) error {
	//format of a: [essid, pass, id]
	//conf, err := generateConfig(a...)
	//if err != nil {
	//      return err
	//}

	//if err := ioutil.WriteFile("/tmp/wifi.conf", conf, 0444); err != nil {
	//      return fmt.Errorf("/tmp/wifi.conf: %v", err)
	//}

	// Each request has a 30 second window to make a connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	//c := make(chan error, 1)

	// There's no telling how long the supplicant will take, but on the other hand,
	// it's been almost instantaneous. But, further, it needs to keep running.
	go func() {
		cmd := exec.CommandContext(ctx, "/usr/bin/strace", "wpa_supplicant", "-i"+w.Interface, "-c/tmp/wifi.conf")
		cmd.Stdout, cmd.Stderr = stdout, stderr
		cmd.Run()
	}()

	// dhclient might never return on incorrect passwords or identity
	go func() {
		cmd := exec.CommandContext(ctx, "dhclient", "-ipv4=true", "-ipv6=false", "-v", w.Interface)
		cmd.Stdout, cmd.Stderr = stdout, stderr
		//      if err := cmd.Run(); err != nil {
		//              c <- err
		//      } else {
		//              c <- nil
		//      }
	}()

	select {
	//case err := <-c:
	//      return err
	case <-ctx.Done():
		return fmt.Errorf("Connection timeout")
	}

}
