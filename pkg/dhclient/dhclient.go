// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dhclient

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/u-root/u-root/pkg/dhclient"
	"github.com/vishvananda/netlink"
)

// Request sets up the dhcp configurations for all of the ifNames.
func Request(ifName string, timeout int, retry int, verbose bool, ipv4 bool, ipv6 bool, cl chan string) {
	ifRE := regexp.MustCompilePOSIX(ifName)

	ifnames, err := netlink.LinkList()
	if err != nil {
		cl <- fmt.Sprintf("Can't get list of link names: %v", err)
		close(cl)
		return
	}

	var filteredIfs []netlink.Link
	for _, iface := range ifnames {
		if ifRE.MatchString(iface.Attrs().Name) {
			filteredIfs = append(filteredIfs, iface)
		}
	}

	if len(filteredIfs) == 0 {
		cl <- fmt.Sprintf("No interfaces match %s", ifName)
		close(cl)
		return
	}

	go configureAll(filteredIfs, cl, timeout, retry, verbose, ipv4, ipv6)

}

func configureAll(ifs []netlink.Link, cl chan<- string, timeout int, retry int, verbose bool, ipv4 bool, ipv6 bool) {
	packetTimeout := time.Duration(timeout) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), packetTimeout*time.Duration(1<<uint(retry)))
	defer cancel()

	c := dhclient.Config{
		Timeout: packetTimeout,
		Retries: retry,
	}
	if verbose {
		c.LogLevel = dhclient.LogSummary
	}
	r := dhclient.SendRequests(ctx, ifs, ipv4, ipv6, c, 30*time.Second)

	defer close(cl)

	for {
		select {
		case <-ctx.Done():
			cl <- fmt.Sprintf("Done with dhclient: %v", ctx.Err())
			return

		case result, ok := <-r:
			if !ok {
				cl <- fmt.Sprintf("Configured all interfaces")
				return
			}
			if result.Err != nil {
				cl <- fmt.Sprintf("Could not configure %s: %v", result.Interface.Attrs().Name, result.Err)
			} else if err := result.Lease.Configure(); err != nil {
				cl <- fmt.Sprintf("Could not configure %s: %v", result.Interface.Attrs().Name, err)
			} else {
				cl <- fmt.Sprintf("Configured %s with %s", result.Interface.Attrs().Name, result.Lease)
				cl <- fmt.Sprintf("Successful")
			}
		}
	}
}
