package main

import (
	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/dhclient"
	"github.com/u-root/webboot/pkg/menu"
)

//	-ifName:  Name of the interface
//	-timeout: Lease timeout in seconds
//	-retry:   Number of DHCP renewals before exiting
//	-verbose: Verbose mode
//	-ipv4:    Use IPV4
//	-ipv6:    Use IPV6
func setUpNetwork(uiEvents <-chan ui.Event) (bool, error) {

	isIfName := func(input string) (string, string, bool) {
		if input[0] == 'e' || input[0] == 'w' {
			return input, "", true
		}
		return "", "not a valid interface name", false
	}

	ifName, err := menu.NewInputWindow("Enter name of the interface:", isIfName, uiEvents)
	if err != nil {
		return false, err
	}

	cl := make(chan string)
	go dhclient.Request(ifName, 15, 5, *v, true, true, cl)
	for {
		msg, ok := <-cl
		if !ok {
			return false, nil
		}
		if msg == "Successful" {
			menu.DisplayResult([]string{msg}, uiEvents)
			return true, nil
		}
		if _, err := menu.DisplayResult([]string{msg}, uiEvents); err != nil {
			return false, err
		}
	}
}
