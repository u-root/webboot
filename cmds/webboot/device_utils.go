package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
)

func getCachedDirectory() (*mount.MountPoint, error) {
	blockDevs, err := block.GetBlockDevices()
	if err != nil {
		log.Fatal("No available block devices to boot from")
	}

	mountPoints, err := ioutil.TempDir("", "temp-device-")
	if err != nil {
		return nil, fmt.Errorf("cannot create tmpdir: %v", err)
	}

	for _, device := range blockDevs {
		mp, err := mount.TryMount("/dev/"+device.Name, filepath.Join(mountPoints, device.Name), "", mount.ReadOnly)
		if err != nil {
			continue
		}
		if _, err = os.Stat(filepath.Join(mp.Path, "Image")); err == nil {
			return mp, nil
		}
	}
	return nil, nil
}
