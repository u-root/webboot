package bootiso

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime/debug"
	"strings"

	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/boot/grub"
	"github.com/u-root/u-root/pkg/boot/kexec"
	"github.com/u-root/u-root/pkg/boot/syslinux"
	"github.com/u-root/u-root/pkg/boot/util"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/u-root/pkg/mount/loop"
	"github.com/u-root/u-root/pkg/uio"
	"golang.org/x/sys/unix"
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
		return nil, fmt.Errorf("Error on ioutil.TempDir; in %s, and got %v", debug.Stack(), err)
	}

	loopdev, err := loop.New(isoPath, "iso9660", "")
	if err != nil {
		return nil, fmt.Errorf("Error on loop.New; in %s, and got %v", debug.Stack(), err)
	}

	mp, err := loopdev.Mount(tmpDir, unix.MS_RDONLY|unix.MS_NOATIME)
	if err != nil {
		return nil, fmt.Errorf("Error on loopdev.Mount; in %s, and got %v", debug.Stack(), err)
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
				return nil, fmt.Errorf("Error on os.Open; in %s, and got %v", debug.Stack(), err)
			}
			files = append(files, kernel)

			// Temp files are not added to the files list
			// since they need to stay open for later reading
			tmpKernel, err = ioutil.TempFile("", "kernel-")
			if err != nil {
				return nil, fmt.Errorf("Error on ioutil.TempFile; in %s, and got %v", debug.Stack(), err)
			}

			if _, err = io.Copy(tmpKernel, kernel); err != nil {
				return nil, fmt.Errorf("Error on io.Copy; in %s, and got %v", debug.Stack(), err)
			}

			if _, err = tmpKernel.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("Error on tmpKernel.Seek; in %s, and got %v", debug.Stack(), err)
			}

			copied[c.KernelPath] = tmpKernel
		} else {
			tmpKernel = copied[c.KernelPath]
		}

		// Copy initrd to temp if we haven't already
		if _, ok := copied[c.InitrdPath]; !ok {
			initrd, err := os.Open(path.Join(tmpDir, c.InitrdPath))
			if err != nil {
				return nil, fmt.Errorf("Error on os.Open; in %s, and got %v", debug.Stack(), err)
			}
			files = append(files, initrd)

			tmpInitrd, err = ioutil.TempFile("", "initrd-")
			if err != nil {
				return nil, fmt.Errorf("Error on ioutil.TempFile; in %s, and got %v", debug.Stack(), err)
			}

			if _, err = io.Copy(tmpInitrd, initrd); err != nil {
				return nil, fmt.Errorf("Error on io.Copy; in %s, and got %v", debug.Stack(), err)
			}

			if _, err = tmpInitrd.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("Error on tmpInitrd.Seek; in %s, and got %v", debug.Stack(), err)
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

// next two functions hoisted from u-root kexec. We will remove
// them when the u-root kexec becomes capable of using the 32-bit
// entry point. 32-bit entry is essential to working on chromebooks.

func copyToFile(r io.Reader) (*os.File, error) {
	f, err := ioutil.TempFile("", "webboot")
	if err != nil {
		return nil, fmt.Errorf("Error on ioutil.TempFile; in %s, and got %v", debug.Stack(), err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return nil, fmt.Errorf("Error on io.Copy; in %s, and got %v", debug.Stack(), err)
	}
	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("Error on f.Sync; in %s, and got %v", debug.Stack(), err)
	}

	readOnlyF, err := os.Open(f.Name())
	if err != nil {
		return nil, fmt.Errorf("Error on os.Open; in %s, and got %v", debug.Stack(), err)
	}
	return readOnlyF, nil
}

// kexecCmd boots via the classic kexec command, if it exists
func cmdKexecLoad(li *boot.LinuxImage, verbose bool) error {
	if li.Kernel == nil {
		return errors.New("LinuxImage.Kernel must be non-nil")
	}

	kernel, initrd := uio.Reader(util.TryGzipFilter(li.Kernel)), uio.Reader(li.Initrd)
	if verbose {
		// In verbose mode, print a dot every 5MiB. It is not pretty,
		// but it at least proves the files are still downloading.
		progress := func(r io.Reader, dot string) io.Reader {
			return &uio.ProgressReadCloser{
				RC:       ioutil.NopCloser(r),
				Symbol:   dot,
				Interval: 5 * 1024 * 1024,
				W:        os.Stdout,
			}
		}
		kernel = progress(kernel, "K")
		initrd = progress(initrd, "I")
	}

	// It seams inefficient to always copy, in particular when the reader
	// is an io.File but that's not sufficient, os.File could be a socket,
	// a pipe or some other strange thing. Also kexec_file_load will fail
	// (similar to execve) if anything as the file opened for writing.
	// That's unfortunately something we can't guarantee here - unless we
	// make a copy of the file and dump it somewhere.
	k, err := copyToFile(kernel)
	if err != nil {
		return err
	}
	defer k.Close()
	kargs := []string{"-d", "-l", "--entry-32bit", "--command-line=" + li.Cmdline}
	var i *os.File
	if li.Initrd != nil {
		i, err = copyToFile(initrd)
		if err != nil {
			return err
		}
		defer i.Close()
		kargs = append(kargs, "--initrd="+i.Name())
	}

	log.Printf("Kernel: %s", k.Name())
	kargs = append(kargs, k.Name())
	if i != nil {
		log.Printf("Initrd: %s", i.Name())
	}
	log.Printf("Command line: %s", li.Cmdline)
	log.Printf("Kexec args: %q", kargs)

	out, err := exec.Command("/sbin/kexec", kargs...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("Load failed; output %q, err %v", out, err)
	}
	return err
}

func cmdKexecReboot(verbose bool) error {
	o, err := exec.Command("/sbin/kexec", "-d", "-e").CombinedOutput()
	if err != nil {
		err = fmt.Errorf("Exec failed; output %q, err %v", o, err)
	}
	return err
}

func BootCachedISO(osImage boot.OSImage, kernelParams string) error {
	// Need to convert from boot.OSImage to boot.LinuxImage to edit the Cmdline
	linuxImage, ok := osImage.(*boot.LinuxImage)
	if !ok {
		return fmt.Errorf("Error converting from boot.OSImage to boot.LinuxImage")
	}

	linuxImage.Cmdline = linuxImage.Cmdline + " " + kernelParams

	// We prefer to use the kexec command for now, if possible, as it can
	// use the 32-bit entry point.
	if _, err := os.Stat("/sbin/kexec"); err != nil {
		if err := cmdKexecLoad(linuxImage, true); err != nil {
			return err
		}
		if err := cmdKexecReboot(true); err != nil {
			return err
		}
	}
	if err := linuxImage.Load(true); err != nil {
		return err
	}

	if err := kexec.Reboot(); err != nil {
		return err
	}

	return nil
}

// VerifyChecksum takes a path to the ISO and its checksum
// and compares the calculated checksum on the ISO against the checksum.
// It returns true if the checksum was correct, false if the checksum
// was incorrect, the calculated checksum, and an error.
func VerifyChecksum(isoPath, checksum, checksumType string) (bool, string, error) {
	iso, err := os.Open(isoPath)
	if err != nil {
		return false, "", err
	}
	defer iso.Close()

	var hash hash.Hash
	switch checksumType {
	case "md5":
		hash = md5.New()
	case "sha1":
		hash = sha1.New()
	case "sha256":
		hash = sha256.New()
	default:
		return false, "", fmt.Errorf("Unknown checksum type.")
	}

	if _, err := io.Copy(hash, iso); err != nil {
		return false, "", err
	}
	calcChecksum := hex.EncodeToString(hash.Sum(nil))

	return calcChecksum == checksum, calcChecksum, nil
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
	devs, err := block.GetBlockDevices()
	if err != nil {
		return nil, fmt.Errorf("Error on block.GetBlockDevices; in %s, and got %v", debug.Stack(), err)
	}
	mp := &mount.Pool{}

	if configType == "syslinux" {
		return syslinux.ParseLocalConfig(context.Background(), mountDir)
	} else if configType == "grub" {
		return grub.ParseLocalConfig(context.Background(), mountDir, devs, mp)
	}

	// If no config type was specified, try both grub and syslinux
	configOpts, err := syslinux.ParseLocalConfig(context.Background(), mountDir)
	if err == nil && len(configOpts) != 0 {
		return configOpts, err
	}
	return grub.ParseLocalConfig(context.Background(), mountDir, devs, mp)
}
