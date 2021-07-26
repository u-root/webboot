// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wifi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/u-root/NiChrome/pkg/wpa/passphrase"
)

const (
	nopassphrase = `network={
		ssid="%s"
		proto=RSN
		key_mgmt=NONE
	}`
	eap = `network={
		ssid="%s"
		key_mgmt=WPA-EAP
		identity="%s"
		password="%s"
	}`
)

var (
	// RegEx for parsing iwlist output
	cellRE       = regexp.MustCompile("(?m)^\\s*Cell")
	essidRE      = regexp.MustCompile("(?m)^\\s*ESSID.*")
	encKeyOptRE  = regexp.MustCompile("(?m)^\\s*Encryption key:(on|off)$")
	wpa2RE       = regexp.MustCompile("(?m)^\\s*IE: IEEE 802.11i/WPA2 Version 1$")
	authSuitesRE = regexp.MustCompile("(?m)^\\s*Authentication Suites .*$")
)

type SecProto int

const (
	NoEnc SecProto = iota
	WpaPsk
	WpaEap
	NotSupportedProto
)

// IWLWorker implements the WiFi interface using the Intel Wireless LAN commands
type IWLWorker struct {
	Interface string
}

func NewIWLWorker(stdout, stderr io.Writer, i string) (WiFi, error) {
	cmd := exec.Command("ip", "link", "set", "dev", i, "up")
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return &IWLWorker{""}, err
	}
	return &IWLWorker{i}, nil
}

func (w *IWLWorker) Scan(stdout, stderr io.Writer) ([]Option, error) {
	// Need a local copy of exec's output to parse out the Iwlist
	var execOutput bytes.Buffer
	stdoutTee := io.MultiWriter(&execOutput, stdout)

	cmd := exec.Command("iwlist", w.Interface, "scanning")
	cmd.Stdout, cmd.Stderr = stdoutTee, stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return parseIwlistOut(execOutput.Bytes()), nil
}

/*
 * Assumptions:
 *	1) Cell, essid, and encryption key option are 1:1 match
 *	2) We only support IEEE 802.11i/WPA2 Version 1
 *	3) Each Wifi only support (1) authentication suites (based on observations)
 */

func parseIwlistOut(o []byte) []Option {
	cells := cellRE.FindAllIndex(o, -1)
	essids := essidRE.FindAll(o, -1)
	encKeyOpts := encKeyOptRE.FindAll(o, -1)

	if cells == nil {
		return nil
	}

	var res []Option
	knownEssids := make(map[string]bool)

	// Assemble all the Wifi options
	for i := 0; i < len(cells); i++ {
		essid := strings.Trim(strings.Split(string(essids[i]), ":")[1], "\"\n")
		if knownEssids[essid] {
			continue
		}
		knownEssids[essid] = true
		encKeyOpt := strings.Trim(strings.Split(string(encKeyOpts[i]), ":")[1], "\n")
		if encKeyOpt == "off" {
			res = append(res, Option{essid, NoEnc})
			continue
		}
		// Find the proper Authentication Suites
		start, end := cells[i][0], len(o)
		if i != len(cells)-1 {
			end = cells[i+1][0]
		}
		// Narrow down the scope when looking for WPA Tag
		wpa2SearchArea := o[start:end]
		l := wpa2RE.FindIndex(wpa2SearchArea)
		if l == nil {
			res = append(res, Option{essid, NotSupportedProto})
			continue
		}
		// Narrow down the scope when looking for Authorization Suites
		authSearchArea := wpa2SearchArea[l[0]:]
		authSuites := strings.Trim(strings.Split(string(authSuitesRE.Find(authSearchArea)), ":")[1], "\n ")
		switch authSuites {
		case "PSK":
			res = append(res, Option{essid, WpaPsk})
		case "802.1x":
			res = append(res, Option{essid, WpaEap})
		default:
			res = append(res, Option{essid, NotSupportedProto})
		}
	}
	return res
}

func (w *IWLWorker) GetID(stdout, stderr io.Writer) (string, error) {
	var execOutput bytes.Buffer
	stdoutTee := io.MultiWriter(&execOutput, stdout)

	cmd := exec.Command("iwgetid", "-r")
	cmd.Stdout, cmd.Stderr = stdoutTee, stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.Trim(execOutput.String(), " \n"), nil
}

func (w *IWLWorker) Connect(stdout, stderr io.Writer, a ...string) error {
	// format of a: [essid, pass, id]
	conf, err := generateConfig(a...)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile("/tmp/wifi.conf", conf, 0444); err != nil {
		var file, err = os.OpenFile("logOutput.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		file.WriteString(time.Now().String() + " ")
		file.WriteString("/temp/wifi.conf: " + err.Error())
		file.WriteString("\n")
		defer file.Close()
		return fmt.Errorf("/tmp/wifi.conf: %v", err)
	}

	// Each request has a 30 second window to make a connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := make(chan error, 1)

	// There's no telling how long the supplicant will take, but on the other hand,
	// it's been almost instantaneous. But, further, it needs to keep running.
	go func() {
		cmd := exec.Command("wpa_supplicant", "-i"+w.Interface, "-c/tmp/wifi.conf")
		outfile, err := os.OpenFile("logOutput.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		cmd.Stdout, cmd.Stderr = outfile, outfile
		if err != nil {
			log.Print(err)
		}
		defer outfile.Close()
		if err = cmd.Run(); err != nil {
			log.Print(err)
			fmt.Sprintf("%s %s\n", time.Now().String(), err.Error())
		}
		c <- fmt.Errorf("wpa supplicant exited unexpectedly")

	}()

	// dhclient might never return on incorrect passwords or identity
	go func() {
		cmd := exec.CommandContext(ctx, "dhclient", "-ipv4=true", "-ipv6=false", "-v", w.Interface)

		outfile, err := os.OpenFile("logOutput.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		cmd.Stdout, cmd.Stderr = outfile, outfile
		if err != nil {
			log.Print(err)
			fmt.Sprintf("%s %s\n", time.Now().String(), err.Error())
		}
		defer outfile.Close()
		c <- cmd.Run()
	}()

	select {
	case err := <-c:
		return err
	case <-ctx.Done():
		return fmt.Errorf("dhcp timeout")
	}
}

func generateConfig(a ...string) (conf []byte, err error) {
	// format of a: [essid, pass, id]
	switch {
	case len(a) == 3:
		conf = []byte(fmt.Sprintf(eap, a[0], a[2], a[1]))
	case len(a) == 2:
		conf, err = passphrase.Run(a[0], a[1])
		if err != nil {
			return nil, fmt.Errorf("essid: %v, pass: %v : %v", a[0], a[1], err)
		}
	case len(a) == 1:
		conf = []byte(fmt.Sprintf(nopassphrase, a[0]))
	default:
		return nil, fmt.Errorf("generateConfig needs 1, 2, or 3 args")
	}
	return
}
