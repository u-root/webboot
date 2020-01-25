// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	debug = func(string, ...interface{}) {}

	verbose = flag.Bool("v", true, "verbose debugging output")
	uroot   = flag.String("u", "", "options for u-root")
	cmds    = flag.String("c", "all", "u-root commands to build into the image")
	wcmds   = flag.String("w", "github.com/u-root/webboot/webboot/.", "webboot commands to build into the image")
	ncmds   = flag.String("n", "github.com/u-root/NiChrome/cmds/wifi", "NiChrome commands to build into the image")
	bzImage = flag.String("bzImage", "", "Optional bzImage to embed in the initramfs")
	iso     = flag.String("iso", "", "Optional iso (e.g. tinycore.iso) to embed in the initramfs")
)

func init() {
	flag.Parse()
	if *verbose {
		debug = log.Printf
	}
}

// This function is a bit nasty but we'll need it until we can extend
// u-root a bit. Consider it a hack to get us ready for OSFC.
// the Must means it has to succeed or we die.
func extraBinMust(n string) string {
	p, err := exec.LookPath(n)
	if err != nil {
		log.Fatalf("extraMustBin(%q): %v", n, err)
	}
	return p
}
func main() {
	var args = []string{
		"go", "run", "github.com/u-root/u-root/.",
		"-files", extraBinMust("iwconfig"),
		"-files", extraBinMust("iwlist"),
		"-files", extraBinMust("wpa_supplicant"),
		"-files", extraBinMust("wpa_action"),
		"-files", extraBinMust("wpa_cli"),
		"-files", extraBinMust("wpa_passphrase"),
	}
	if *bzImage != "" {
		args = append(args, "-files", *bzImage+":bzImage")
	}
	if *iso != "" {
		args = append(args, "-files", *iso+":iso")
	}
	var commands = [][]string{
		{"date"},
		{"go", "get", "-u", "github.com/u-root/u-root"},
		{"go", "get", "-d", "-v", "-u", "github.com/u-root/NiChrome/..."},
		append(append(args, strings.Fields(*uroot)...), *cmds, *wcmds, *ncmds),
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
