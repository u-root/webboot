package webboot

//Distro defines an operating system distribution
type Distro struct {
	Kernel       string
	Initrd       string
	Cmdline      string
	DownloadLink string
}
