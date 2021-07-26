// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main takes the connect function of iwl.go and reduces it
// to as few lines as possible to make spotting errors in wifi bugs easier.
package main

import (
	"fmt"
	"os"
	"os/exec"
)

// IWLWorker implements the WiFi interface using the Intel Wireless LAN commands
type IWLWorker struct {
	Interface string
}

func main() {
	// There's no telling how long the supplicant will take, but on the other hand,
	// it's been almost instantaneous. But, further, it needs to keep running.
	go func() {
		cmd := exec.Command("/usr/bin/strace", "-o", "/tmp/out", "wpa_supplicant", "-dd", "-i    wlan0", "-c/tmp/wifi.conf")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Println("wpa supplicant cmd.Stdout: ", cmd.Stdout)
		fmt.Println("wpa supplicant cmd.Stderr: ", cmd.Stderr)
		err := cmd.Run()
		fmt.Println("wpa supplicant cmd.Run() error: ", err)
	}()

	// dhclient might never return on incorrect passwords or identity
	go func() {
		cmd := exec.Command("dhclient", "-ipv4=true", "-ipv6=false", "-v", "wlan0")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		fmt.Println("dhclient cmd.Stdout: ", cmd.Stdout)
		fmt.Println("dhclient cmd.Stderr: ", cmd.Stderr)
		err := cmd.Run()
		fmt.Println("dhclient cmd.Run() error: ", err)
	}()

	for {
	}
}
