// Copyright 2019-2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

// This program depends on the presence of the u-root project.
// First time use requires that you run
// go get -u github.com/u-root/u-root
import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type cmd struct {
	args []string
	dir  string
}

var (
	debug = func(string, ...interface{}) {}

	verbose    = flag.Bool("v", true, "verbose debugging output")
	uroot      = flag.String("u", "", "options for u-root")
	cmds       = flag.String("c", "", "u-root commands to build into the image")
	bzImage    = flag.String("bzImage", "", "Optional bzImage to embed in the initramfs")
	iso        = flag.String("iso", "", "Optional iso (e.g. tinycore.iso) to embed in the initramfs")
	wifi       = flag.Bool("wifi", true, "include wifi tools")
	wpaVersion = flag.String("wpa-version", "system", "if set, download and build the wpa_supplicant (ex: 2.9)")
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
	debug("Using %q from %q", n, p)
	return p
}

// buildWPASupplicant downloads and builds the wpa_supplicant (and other tools)
// statically. The path containing these tools is returned.
func buildWPASupplicant(version string) (string, error) {
	// Download and extract the tar release.
	url := fmt.Sprintf("https://w1.fi/releases/wpa_supplicant-%s.tar.gz", version)
	file := fmt.Sprintf("wpa_supplicant-%s.tar.gz", version)
	extractDir := fmt.Sprintf("wpa_supplicant-%s", version)
	workDir := filepath.Join(extractDir, "wpa_supplicant")
	if _, err := os.Stat(extractDir); os.IsNotExist(err) {
		// Download file.
		if err := downloadFile(url, file); err != nil {
			return "", err
		}

		// Extract the tar file.
		debug("Extracting %q to %q...", file, extractDir)
		tarArgs := []string{"-x", "-f", file}
		if *verbose {
			tarArgs = append(tarArgs, "-v")
		}
		cmd := exec.Command("tar", tarArgs...)
		if *verbose {
			cmd.Stdout = os.Stdout
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("Failed to extract %q: %v", file, err)
		}
	} else if err == nil {
		log.Printf("Directory %q already exists, skipping download", extractDir)
	} else {
		return "", fmt.Errorf("error with stat on %q: %v", extractDir, err)
	}

	wantFiles := []string{
		filepath.Join(workDir, "wpa_supplicant"),
		filepath.Join(workDir, "wpa_cli"),
		filepath.Join(workDir, "wpa_passphrase"),
	}
	if err := checkFilesExist(wantFiles); err == nil {
		log.Printf("Files %v already exist, skipping build", wantFiles)
	} else {
		debug("Building wpa_supplicant...")
		// Use the defconfig. Everything related to DBUS is stripped out
		// because DBUS breaks the static build.
		origin := filepath.Join(workDir, "defconfig")
		destination := filepath.Join(workDir, ".config")
		if err := filterFile(origin, destination, regexp.MustCompile("DBUS")); err != nil {
			return "", fmt.Errorf("error creating .config: %v", err)
		}

		// Build with the following options:
		//   -Os: Optimize for size
		//   -flto: Link time optimization (reduces size)
		//   -static: No dynamic dependencies
		//   -pthread: Use gcc's version of pthreads to be static
		//   -s: Strip symbols
		cmd := exec.Command("make", fmt.Sprintf("-j%d", runtime.NumCPU()), "EXTRA_CFLAGS=-Os -flto", "LDFLAGS=-static -pthread -Os -flto -s")
		cmd.Dir = workDir
		debug("cd %q && %s", workDir, cmd)
		if *verbose {
			cmd.Stdout = os.Stdout
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to compile wpa_supplicant: %v", err)
		}

		if err := checkFilesExist(wantFiles); err != nil {
			return "", fmt.Errorf("failed to build files %v, they do not exist: %v", wantFiles, err)
		}
	}

	return workDir, nil
}

// downloadFile download from the given url to the given file.
func downloadFile(url, file string) error {
	debug("Downloading %q to %q...", url, file)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading %q: %s", url, resp.Status)
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// checkFilesExist if each file exist.
func checkFilesExist(files []string) error {
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return fmt.Errorf("%q does not exist", f)
		} else if err != nil {
			return fmt.Errorf("error with stat on %q: %v", f, err)
		}
	}
	return nil
}

// filterFile copies a file from origin to destination while deleting matching lines.
func filterFile(origin, destination string, filterOut *regexp.Regexp) error {
	// Open the files.
	originF, err := os.Open(origin)
	if err != nil {
		return err
	}
	defer originF.Close()
	destF, err := os.Create(destination)
	if err != nil {
		return err
	}

	// Copy the lines.
	s := bufio.NewScanner(originF)
	for s.Scan() {
		line := s.Text()
		if !filterOut.MatchString(line) {
			if _, err := destF.WriteString(line + "\n"); err != nil {
				destF.Close()
				return err
			}
		}
	}
	if err := s.Err(); err != nil {
		destF.Close()
		return err
	}
	return destF.Close()
}

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current directory %v", err)
	}

	if _, err := os.Stat("u-root"); err != nil {
		c := exec.Command("git", "clone", "--single-branch", "https://github.com/u-root/u-root")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			log.Fatalf("cloning u-root: %v", err)
		}
	}

	// Use the system wpa_supplicant or download them.
	if *wpaVersion != "system" {
		wpaSupplicantPath, err := buildWPASupplicant(*wpaVersion)
		if err != nil {
			log.Fatalf("Error building wpa_supplicant: %v", err)
		}
		// Add to front of PATH to be picked up later.
		if err := os.Setenv("PATH", fmt.Sprintf("%s:%s", wpaSupplicantPath, os.Getenv("PATH"))); err != nil {
			log.Fatalf("Error setting PATH env variable: %v", err)
		}
	}

	var args = []string{
		"u-root", "-files", "/etc/ssl/certs", "-uroot-source=./u-root/",
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
			"-files", extraBinMust("wpa_supplicant")+":bin/wpa_supplicant",
			"-files", extraBinMust("wpa_cli")+":bin/wpa_cli",
			"-files", extraBinMust("wpa_passphrase")+":bin/wpa_passphrase",
			"-files", extraBinMust("strace"),
			"-files", "cmds/webboot/distros.json:distros.json",
		)
	}
	if *bzImage != "" {
		args = append(args, "-files", *bzImage+":bzImage")
	}
	if *iso != "" {
		args = append(args, "-files", *iso+":iso")
	}
	args = append(args, "core", "cmds/*")
	var commands = []cmd{
		{args: []string{"go", "build"}, dir: filepath.Join(currentDir, "cmds", "webboot")},
		{args: append(append(args, strings.Fields(*uroot)...), *cmds)},
	}

	for _, cmd := range commands {
		debug("Run %v", cmd)
		c := exec.Command(cmd.args[0], cmd.args[1:]...)
		c.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		c.Dir = cmd.dir
		if err := c.Run(); err != nil {
			log.Fatalf("%s failed: %v", cmd, err)
		}
	}
	debug("done")
}
