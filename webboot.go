// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

// This program depends on the present of the u-root and NiChrome projects.
// First time use requires that you run
// go get -u github.com/u-root/u-root
// go get -u github.com/u-root/NiChrome/...
// We no longer fetch them here automagically because it makes offline
// use painful.

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type cmd struct {
	args []string
	dir  string
}

var (
	debug = func(string, ...interface{}) {}

	verbose = flag.Bool("v", true, "verbose debugging output")
	uroot   = flag.String("u", "", "options for u-root")
	cmds    = flag.String("c", "all", "u-root commands to build into the image")
	ncmds   = flag.String("n", "github.com/u-root/NiChrome/cmds/wifi", "NiChrome commands to build into the image")
	bzImage = flag.String("bzImage", "", "Optional bzImage to embed in the initramfs")
	iso     = flag.String("iso", "", "Optional iso (e.g. tinycore.iso) to embed in the initramfs")
	wifi    = flag.Bool("wifi", true, "include wifi tools")
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

	currentDir, err := os.Getwd()

	if err != nil {
		log.Fatalf("error getting current directory %v", err)
	}
	var args = []string{
		"go", "run", "github.com/u-root/u-root/.",
		"-files", "/etc/ssl/certs",
	}

	// Try to find the system kexec. We can not use LookPath as people
	// building this might have the u-root kexec in their path.
	if _, err := os.Stat("/sbin/kexec"); err == nil {
		args = append(args, "-files=/sbin/kexec")
	}

	if *wifi {
		args = append(args,
			"-files", extraBinMust("iwconfig"),
			"-files", extraBinMust("iwlist"),
			"-files", extraBinMust("wpa_supplicant"),
			"-files", extraBinMust("wpa_action"),
			"-files", extraBinMust("wpa_cli"),
			"-files", extraBinMust("wpa_passphrase"),
			"-files", filepath.Join(currentDir, "cmds", "webboot", "webboot")+":bbin/webboot")
	}
	if *bzImage != "" {
		args = append(args, "-files", *bzImage+":bzImage")
	}
	if *iso != "" {
		args = append(args, "-files", *iso+":iso")
	}
	var commands = []cmd{
		{args: []string{"date"}},
		{args: []string{"ls"}},
		{args: []string{"pwd"}},
		// {args: []string{"go", "build"}, dir: filepath.Join(currentDir, "cmds", "oldwebboot")},
		{args: []string{"go", "build"}, dir: filepath.Join(currentDir, "cmds", "webboot")},
		{args: append(append(args, strings.Fields(*uroot)...), *cmds, *ncmds)},
	}

	for _, cmd := range commands {
		debug("Run %v", cmd)
		c := exec.Command(cmd.args[0], cmd.args[1:]...)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		c.Dir = cmd.dir
		if err := c.Run(); err != nil {
			log.Fatalf("%s failed: %v", cmd, err)
		}
	}
	debug("done")
}
