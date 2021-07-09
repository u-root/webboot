package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
	"github.com/u-root/webboot/pkg/wifi"
	"github.com/vishvananda/netlink"
)

// Collect stdout and stderr from the network setup.
// Declare globally because wifi.Connect() triggers
// go routines that might still be running after return.
var wifiStdout, wifiStderr bytes.Buffer

func connected() bool {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	if _, err := client.Get("http://google.com"); err != nil {
		return false
	}
	return true
}

func wirelessIfaceEntries() ([]menu.Entry, error) {
	interfaces, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	var ifEntries []menu.Entry
	for _, iface := range interfaces {
		if interfaceIsWireless(iface.Attrs().Name) {
			ifEntries = append(ifEntries, &Interface{label: iface.Attrs().Name})
		}
	}
	return ifEntries, nil
}

func interfaceIsWireless(ifname string) bool {
	devPath := fmt.Sprintf("/sys/class/net/%s/wireless", ifname)
	if _, err := os.Stat(devPath); err != nil {
		return false
	}
	return true
}

func setupNetwork(uiEvents <-chan ui.Event, menus chan<- string) error {
	iface, err := selectNetworkInterface(uiEvents, menus)
	if err != nil {
		return err
	}

	return selectWirelessNetwork(uiEvents, menus, iface.Label())
}

func selectNetworkInterface(uiEvents <-chan ui.Event, menus chan<- string) (menu.Entry, error) {
	ifEntries, err := wirelessIfaceEntries()
	if err != nil {
		return nil, err
	}

	iface, err := menu.PromptMenuEntry("Network Interfaces", "Choose an option", ifEntries, uiEvents, menus)
	if err != nil {
		return nil, err
	}

	return iface, nil
}

func selectWirelessNetwork(uiEvents <-chan ui.Event, menus chan<- string, iface string) error {
	worker, err := wifi.NewIWLWorker(&wifiStdout, &wifiStderr, iface)
	if err != nil {
		return err
	}

	for {
		progress := menu.NewProgress("Scanning for wifi networks", true)
		networkScan, err := worker.Scan(&wifiStdout, &wifiStderr)
		progress.Close()
		if err != nil {
			return err
		}

		netEntries := []menu.Entry{}
		for _, network := range networkScan {
			netEntries = append(netEntries, &Network{info: network})
		}

		entry, err := menu.PromptMenuEntry("Wireless Networks", "Choose an option", netEntries, uiEvents, menus)
		if err != nil {
			return err
		}

		network, ok := entry.(*Network)
		if !ok {
			return fmt.Errorf("Bad menu entry.")
		}

		if err := connectWirelessNetwork(uiEvents, menus, worker, network.info); err != nil {
			switch err {
			case menu.ExitRequest: // user typed <Ctrl+d> to exit
				return err
			case menu.BackRequest: // user typed <Esc> to go back
				continue
			default: // connection error
				menu.DisplayResult([]string{err.Error()}, uiEvents, menus)
				continue
			}
		}

		return nil
	}
}

func connectWirelessNetwork(uiEvents <-chan ui.Event, menus chan<- string, worker wifi.WiFi, network wifi.Option) error {
	var setupParams = []string{network.Essid}
	authSuite := network.AuthSuite

	if authSuite == wifi.NotSupportedProto {
		return fmt.Errorf("Security protocol is not supported.")
	} else if authSuite == wifi.WpaPsk || authSuite == wifi.WpaEap {
		credentials, err := enterCredentials(uiEvents, menus, authSuite)
		if err != nil {
			return err
		}
		setupParams = append(setupParams, credentials...)
	}

	progress := menu.NewProgress("Connecting to network", true)
	err := worker.Connect(&wifiStdout, &wifiStderr, setupParams...)
	progress.Close()
	if err != nil {
		return err
	}

	return nil
}

func enterCredentials(uiEvents <-chan ui.Event, menus chan<- string, authSuite wifi.SecProto) ([]string, error) {
	var credentials []string
	pass, err := menu.PromptTextInput("Enter password:", menu.AlwaysValid, uiEvents, menus)
	if err != nil {
		return nil, err
	}

	credentials = append(credentials, pass)
	if authSuite == wifi.WpaPsk {
		return credentials, nil
	}

	// If not WpaPsk, the network uses WpaEap and also needs an identity
	identity, err := menu.PromptTextInput("Enter identity:", menu.AlwaysValid, uiEvents, menus)
	if err != nil {
		return nil, err
	}

	credentials = append(credentials, identity)
	return credentials, nil
}
