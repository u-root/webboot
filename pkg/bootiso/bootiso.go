package bootiso

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/boot/grub"
	"github.com/u-root/u-root/pkg/boot/kexec"
	"github.com/u-root/u-root/pkg/boot/syslinux"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
	"golang.org/x/sys/unix"
)

var (
	SupportedChecksums = []string{"md5", "sha256"}
)

type Config struct {
	Label      string
	KernelPath string
	InitrdPath string
	Cmdline    string
}

// ParseConfigFromISO mounts the iso file, attempts to parse the config file,
// and returns a list of bootable boot.OSImage objects representing the parsed configs
func ParseConfigFromISO(isoPath string, configType string) ([]boot.OSImage, error) {
	tmp, err := ioutil.TempDir("", "mnt-")
	if err != nil {
		return nil, fmt.Errorf("Error creating mount dir: %v", err)
	}
	defer os.RemoveAll(tmp)

	loopdev, err := loop.New(isoPath, "iso9660", "")
	if err != nil {
		return nil, fmt.Errorf("Error creating loop device: %v", err)
	}

	mp, err := loopdev.Mount(tmp, unix.MS_RDONLY|unix.MS_NOATIME)
	if err != nil {
		return nil, fmt.Errorf("Error mounting loop device: %v", err)
	}
	defer mp.Unmount(0)

	images, err := parseConfigFile(tmp, configType)
	if err != nil {
		return nil, fmt.Errorf("Error parsing config: %v", err)
	}

	return images, nil
}

// LoadCustomConfigs is an alternative to ParseConfigFromISO that allows us
// to define the boot parameters ourselves (in a list of Config objects)
// instead of parsing them from a config file
func LoadCustomConfigs(isoPath string, configs []Config) ([]boot.OSImage, error) {
	tmpDir, err := ioutil.TempDir("", "mnt-")
	if err != nil {
		return nil, err
	}

	loopdev, err := loop.New(isoPath, "iso9660", "")
	if err != nil {
		return nil, err
	}

	mp, err := loopdev.Mount(tmpDir, unix.MS_RDONLY|unix.MS_NOATIME)
	if err != nil {
		return nil, err
	}

	var images []boot.OSImage
	var files []*os.File
	copied := make(map[string]*os.File)

	defer func() {
		for _, f := range files {
			if err = f.Close(); err != nil {
				log.Print(err)
			}
		}

		if err = mp.Unmount(unix.MNT_FORCE); err != nil {
			log.Fatal(err)
		}

		// Use Remove rather than RemoveAll to avoid
		// removal if the directory is not empty
		if err = os.Remove(tmpDir); err != nil {
			log.Fatal(err)
		}
	}()

	for _, c := range configs {
		var tmpKernel, tmpInitrd *os.File

		// Copy kernel to temp if we haven't already
		if _, ok := copied[c.KernelPath]; !ok {
			kernel, err := os.Open(path.Join(tmpDir, c.KernelPath))
			if err != nil {
				return nil, err
			}
			files = append(files, kernel)

			// Temp files are not added to the files list
			// since they need to stay open for later reading
			tmpKernel, err = ioutil.TempFile("", "kernel-")
			if err != nil {
				return nil, err
			}

			if _, err = io.Copy(tmpKernel, kernel); err != nil {
				return nil, err
			}

			if _, err = tmpKernel.Seek(0, 0); err != nil {
				return nil, err
			}

			copied[c.KernelPath] = tmpKernel
		} else {
			tmpKernel = copied[c.KernelPath]
		}

		// Copy initrd to temp if we haven't already
		if _, ok := copied[c.InitrdPath]; !ok {
			initrd, err := os.Open(path.Join(tmpDir, c.InitrdPath))
			if err != nil {
				return nil, err
			}
			files = append(files, initrd)

			tmpInitrd, err = ioutil.TempFile("", "initrd-")
			if err != nil {
				return nil, err
			}

			if _, err = io.Copy(tmpInitrd, initrd); err != nil {
				return nil, err
			}

			if _, err = tmpInitrd.Seek(0, 0); err != nil {
				return nil, err
			}

			copied[c.InitrdPath] = tmpInitrd
		} else {
			tmpInitrd = copied[c.InitrdPath]
		}

		images = append(images, &boot.LinuxImage{
			Name:    c.Label,
			Kernel:  tmpKernel,
			Initrd:  tmpInitrd,
			Cmdline: c.Cmdline,
		})
	}

	return images, nil
}

// BootFromPmem copies the ISO to pmem0 and boots
// given the syslinux configuration with the provided label
func BootFromPmem(isoPath string, configLabel string, configType string) error {
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
	defer os.RemoveAll(tmp)

	if _, err := mount.Mount("/dev/pmem0", tmp, "iso9660", "", unix.MS_RDONLY|unix.MS_NOATIME); err != nil {
		return fmt.Errorf("Error mounting pmem0 to temp directory: %v", err)
	}

	configOpts, err := parseConfigFile(tmp, configType)
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

func BootCachedISO(osImage boot.OSImage, kernelParams string) error {
	// Need to convert from boot.OSImage to boot.LinuxImage to edit the Cmdline
	linuxImage, ok := osImage.(*boot.LinuxImage)
	if !ok {
		return fmt.Errorf("Error converting from boot.OSImage to boot.LinuxImage")
	}

	linuxImage.Cmdline = linuxImage.Cmdline + " " + kernelParams

	if err := linuxImage.Load(true); err != nil {
		return err
	}

	if err := kexec.Reboot(); err != nil {
		return err
	}

	return nil
}

// VerifyChecksum takes a path to the ISO and its checksum file
// and compares the calculated checksum on the ISO
// against the value parsed from the checksum file
func VerifyChecksum(isoPath, checksumPath, checksumType string) (bool, error) {
	iso, err := os.Open(isoPath)
	if err != nil {
		return false, err
	}
	defer iso.Close()

	checksumFile, err := os.Open(checksumPath)
	if err != nil {
		return false, err
	}
	defer checksumFile.Close()

	var hash hash.Hash
	switch checksumType {
	case "md5":
		hash = md5.New()
	case "sha256":
		hash = sha256.New()
	default:
		return false, fmt.Errorf("Unknown checksum type.")
	}

	if _, err := io.Copy(hash, iso); err != nil {
		return false, err
	}
	calcChecksum := hex.EncodeToString(hash.Sum(nil))

	var parsedChecksum string
	isoName := path.Base(isoPath)
	scanner := bufio.NewScanner(checksumFile)

	// Checksum file should contain a line with
	// the checksum and the ISO file name
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, isoName) {
			splitLine := strings.Split(line, " ")
			parsedChecksum = splitLine[0]
			break
		}
	}

	return calcChecksum == parsedChecksum, nil
}

func findConfigOptionByLabel(configOptions []boot.OSImage, configLabel string) boot.OSImage {
	for _, config := range configOptions {
		if config.Label() == configLabel {
			return config
		}
	}
	return nil
}

func parseConfigFile(mountDir string, configType string) ([]boot.OSImage, error) {
	if configType == "syslinux" {
		return syslinux.ParseLocalConfig(context.Background(), mountDir)
	} else if configType == "grub" {
		return grub.ParseLocalConfig(context.Background(), mountDir)
	}

	// If no config type was specified, try both grub and syslinux
	configOpts, err := syslinux.ParseLocalConfig(context.Background(), mountDir)
	if err == nil && len(configOpts) != 0 {
		return configOpts, err
	}
	return grub.ParseLocalConfig(context.Background(), mountDir)
}
