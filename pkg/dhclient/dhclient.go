// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dhclient

import (
	"context"
	"log"
	"regexp"
	"time"

	"github.com/u-root/u-root/pkg/dhclient"
	"github.com/vishvananda/netlink"
)

// Request sets up the dhcp configurations for all of the ifNames.
func Request(ifName string, timeout int, retry int, verbose bool, ipv4 bool, ipv6 bool) {
	ifRE := regexp.MustCompilePOSIX(ifName)

	ifnames, err := netlink.LinkList()
	if err != nil {
		log.Fatalf("Can't get list of link names: %v", err)
	}

	var filteredIfs []netlink.Link
	for _, iface := range ifnames {
		if ifRE.MatchString(iface.Attrs().Name) {
			filteredIfs = append(filteredIfs, iface)
		}
	}

	if len(filteredIfs) == 0 {
		log.Fatalf("No interfaces match %s", ifName)
	}

	cl := make(chan error)
	go configureAll(filteredIfs, cl, timeout, retry, verbose, ipv4, ipv6)
	result := <-cl
	log.Printf("Configuring DHCP returns with error: %v", result)
}

func configureAll(ifs []netlink.Link, cl chan<- error, timeout int, retry int, verbose bool, ipv4 bool, ipv6 bool) {
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
			log.Printf("Done with dhclient: %v", ctx.Err())
			return

		case result, ok := <-r:
			if !ok {
				log.Printf("Configured all interfaces.")
				return
			}
			if result.Err != nil {
				log.Printf("Could not configure %s: %v", result.Interface.Attrs().Name, result.Err)
			} else if err := result.Lease.Configure(); err != nil {
				log.Printf("Could not configure %s: %v", result.Interface.Attrs().Name, err)
			} else {
				cl <- nil
				log.Printf("Configured %s with %s", result.Interface.Attrs().Name, result.Lease)
			}
		}
	}
}
