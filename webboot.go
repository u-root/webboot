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
		{"go", "run", "github.com/u-root/u-root/.", *uroot, *cmds},
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
