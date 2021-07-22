// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// IWLWorker implements the WiFi interface using the Intel Wireless LAN commands
type IWLWorker struct {
	Interface string
}

func main() {
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
		cmd := exec.CommandContext(ctx, "/usr/bin/strace", "-o", "/tmp/out", "wpa_supplicant", "-dd", "-iwlan0", "-c/tmp/wifi.conf")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Println("wpa supplicant cmd.Stdout: ", cmd.Stdout)
		fmt.Println("wpa supplicant cmd.Stderr: ", cmd.Stderr)
		err := cmd.Run()
		fmt.Println("wpa supplicant cmd.Run() error: ", err)
	}()

	// dhclient might never return on incorrect passwords or identity
	go func() {
		cmd := exec.CommandContext(ctx, "dhclient", "-ipv4=true", "-ipv6=false", "-v", "wlan0")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Println("dhclient cmd.Stdout: ", cmd.Stdout)
		fmt.Println("dhclient cmd.Stderr: ", cmd.Stderr)
		err := cmd.Run()
		fmt.Println("dhclient cmd.Run() error: ", err)
		//      if err := cmd.Run(); err != nil {
		//              c <- err
		//      } else {
		//              c <- nil
		//      }
	}()

	//select {
	//case err := <-c:
	//      return err
	//case <-ctx.Done():
	//	log.Fatalf("Connection timeout")
	//}
	for {
	}
}
