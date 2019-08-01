package webboot

import (
	"log"

	"github.com/u-root/u-root/pkg/cmdline"
)

//Distro defines an operating system distribution
type Distro struct {
	Kernel       string
	Initrd       string
	Cmdline      string
	DownloadLink string
}

//CommandLine processes the command line arguments to webboot
func CommandLine(distro Distro, commandln string) {
	distro.Cmdline = commandln
	if distro.Cmdline == "reuse-cmdline" {
		procCmdLine := cmdline.NewCmdLine()
		if procCmdLine.Err != nil {
			log.Fatal("Couldn't read /proc/cmdline")
		} else {
			distro.Cmdline = procCmdLine.Raw
		}
	}
}
