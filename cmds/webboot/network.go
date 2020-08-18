package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/NiChrome/pkg/wifi"
	"github.com/u-root/webboot/pkg/menu"
	"github.com/vishvananda/netlink"
)

func connected() bool {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	if _, err := client.Get("http://google.com"); err != nil {
		return false
	}
	return true
}

func interfaceEntries() ([]menu.Entry, error) {
	interfaces, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	var ifEntries []menu.Entry
	for _, iface := range interfaces {
		ifEntries = append(ifEntries, &Interface{label: iface.Attrs().Name})
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

func setupNetwork(uiEvents <-chan ui.Event) error {
	iface, err := selectNetworkInterface(uiEvents)
	if err != nil {
		return err
	}

	return selectWirelessNetwork(uiEvents, iface)
}

func selectNetworkInterface(uiEvents <-chan ui.Event) (string, error) {
	ifEntries, err := interfaceEntries()
	if err != nil {
		return "", err
	}

	for {
		iface, err := menu.DisplayMenu("Network Interfaces", "Choose an option", ifEntries, uiEvents)
		if err != nil {
			return "", err
		}

		if !interfaceIsWireless(iface.Label()) {
			menu.DisplayResult([]string{"Only wireless network interfaces are supported."}, uiEvents)
		} else {
			return iface.Label(), nil
		}
	}
}

func selectWirelessNetwork(uiEvents <-chan ui.Event, iface string) error {
	worker, err := wifi.NewIWLWorker(iface)
	if err != nil {
		return err
	}

	for {
		networkScan, err := worker.Scan()
		if err != nil {
			return err
		}

		netEntries := []menu.Entry{}
		for _, network := range networkScan {
			netEntries = append(netEntries, &Network{info: network})
		}

		entry, err := menu.DisplayMenu("Wireless Networks", "Choose an option", netEntries, uiEvents)
		if err != nil {
			return err
		}

		network, ok := entry.(*Network)
		if !ok {
			return fmt.Errorf("Bad menu entry.")
		}

		if err := connectWirelessNetwork(uiEvents, worker, network.info); err != nil {
			menu.DisplayResult([]string{err.Error()}, uiEvents)
			continue
		}

		return nil
	}
}

func connectWirelessNetwork(uiEvents <-chan ui.Event, worker wifi.WiFi, network wifi.Option) error {
	var setupParams = []string{network.Essid}
	authSuite := network.AuthSuite

	if authSuite == wifi.NotSupportedProto {
		return fmt.Errorf("Security protocol is not supported.")
	} else if authSuite == wifi.WpaPsk || authSuite == wifi.WpaEap {
		credentials, err := enterCredentials(uiEvents, authSuite)
		if err != nil {
			return err
		}
		setupParams = append(setupParams, credentials...)
	}

	if err := worker.Connect(setupParams...); err != nil {
		return err
	}

	return nil
}

func enterCredentials(uiEvents <-chan ui.Event, authSuite wifi.SecProto) ([]string, error) {
	var credentials []string
	pass, err := menu.NewInputWindow("Enter password:", menu.AlwaysValid, uiEvents)
	if err != nil {
		return nil, err
	}

	credentials = append(credentials, pass)
	if authSuite == wifi.WpaPsk {
		return credentials, nil
	}

	// If not WpaPsk, the network uses WpaEap and also needs an identity
	identity, err := menu.NewInputWindow("Enter identity:", menu.AlwaysValid, uiEvents)
	if err != nil {
		return nil, err
	}

	credentials = append(credentials, identity)
	return credentials, nil
}
