// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
)

var (
	debug   = func(string, ...interface{}) {}
	verbose = flag.Bool("v", true, "verbose debugging output")
	uroot   = flag.String("u", "", "options for u-root")
	cmds    = flag.String("c", "core", "u-root commands to build into the image")
	wcmds   = flag.String("w", "github.com/u-root/webboot/webboot/.", "webboot commands to build into the image")
	ncmds   = flag.String("n", "github.com/u-root/NiChrome/cmds/wifi", "NiChrome commands to build into the image")
)

func init() {
	flag.Parse()
	if *verbose {
		debug = log.Printf
	}
}

func main() {
	var commands = [][]string{
		{"date"},
		{"go", "get", "-u", "github.com/u-root/u-root"},
		{"go", "get", "-d", "-v", "-u", "github.com/u-root/NiChrome/..."},
		{"sudo", "apt", "install", "wireless-tools"},
		{"go", "run", "github.com/u-root/u-root/.", "-files", "/sbin/iwconfig:bin/iwconfig", "-files", "/sbin/iwlist:bin/iwlist", *uroot, *cmds, *wcmds, *ncmds},
	}
	for _, cmd := range commands {
		debug("Run %v", cmd)
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			log.Fatalf("%s failed: %v", cmd, err)
		}
	}
	debug("done")
}
