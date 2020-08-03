package bootiso

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/boot/kexec"
	"github.com/u-root/u-root/pkg/boot/syslinux"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
	"golang.org/x/sys/unix"
)

// ParseConfigFromISO mounts an iso file to a
// temp dir to get the config options
func ParseConfigFromISO(isoPath string) ([]boot.OSImage, error) {
	tmp, err := ioutil.TempDir("", "mnt")
	if err != nil {
		return nil, fmt.Errorf("Error creating mount dir: %v", err)
	}

	loopdev, err := loop.New(isoPath, "iso9660", "")
	if err != nil {
		return nil, fmt.Errorf("Error creating loop device: %v", err)
	}

	mp, err := loopdev.Mount(tmp, unix.MS_RDONLY|unix.MS_NOATIME)
	if err != nil {
		return nil, fmt.Errorf("Error mounting loop device: %v", err)
	}
	defer mp.Unmount(0)

	configOpts, err := syslinux.ParseLocalConfig(context.Background(), tmp)
	if err != nil {
		return nil, fmt.Errorf("Error parsing config: %v", err)
	}

	return configOpts, nil
}

// BootFromPmem copies the ISO to pmem0 and boots
// given the syslinux configuration with the provided label
func BootFromPmem(isoPath string, configLabel string) error {
	pmem, err := os.OpenFile("/dev/pmem0", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("Error opening persistent memory device: %v", err)
	}

	iso, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("Error opening ISO: %v", err)
	}
	defer iso.Close()

	if _, err := io.Copy(pmem, iso); err != nil {
		return fmt.Errorf("Error copying from ISO to pmem: %v", err)
	}
	if err = pmem.Close(); err != nil {
		return fmt.Errorf("Error closing persistent memory device: %v", err)
	}

	tmp, err := ioutil.TempDir("", "mnt")
	if err != nil {
		return fmt.Errorf("Error creating temp directory: %v", err)
	}

	if _, err := mount.Mount("/dev/pmem0", tmp, "iso9660", "", unix.MS_RDONLY|unix.MS_NOATIME); err != nil {
		return fmt.Errorf("Error mounting pmem0 to temp directory: %v", err)
	}

	configOpts, err := syslinux.ParseLocalConfig(context.Background(), tmp)
	if err != nil {
		return fmt.Errorf("Error retrieving syslinux config options: %v", err)
	}

	osImage := findConfigOptionByLabel(configOpts, configLabel)
	if osImage == nil {
		return fmt.Errorf("Config option with the requested label does not exist")
	}

	// Need to convert from boot.OSImage to boot.LinuxImage to edit the Cmdline
	linuxImage, ok := osImage.(*boot.LinuxImage)
	if !ok {
		return fmt.Errorf("Error converting from boot.OSImage to boot.LinuxImage")
	}

	localCmd, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		return fmt.Errorf("Error accessing /proc/cmdline")
	}
	cmdline := strings.TrimSuffix(string(localCmd), "\n") + " " + linuxImage.Cmdline
	linuxImage.Cmdline = cmdline

	if err := linuxImage.Load(true); err != nil {
		return err
	}
	if err := kexec.Reboot(); err != nil {
		return err
	}

	return nil
}

func findConfigOptionByLabel(configOptions []boot.OSImage, configLabel string) boot.OSImage {
	for _, config := range configOptions {
		if config.Label() == configLabel {
			return config
		}
	}
	return nil
}
