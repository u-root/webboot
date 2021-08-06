// Package main downloads and boots an ISO.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	Boot "github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/webboot/pkg/bootiso"
	"github.com/u-root/webboot/pkg/menu"
)

var (
	v          = flag.Bool("verbose", false, "Verbose output")
	verbose    = func(string, ...interface{}) {}
	dir        = flag.String("dir", "", "Path of cached directory")
	dryRun     = flag.Bool("dryrun", false, "If dry_run is true we won't boot the iso.")
	distroName = flag.String("distroName", "", "This is the distro that will be tested.")
	cacheDev   CacheDevice
	logBuffer  bytes.Buffer
	tmpBuffer  bytes.Buffer
)

// ISO's exec downloads the iso and boot it.
func (i *ISO) exec(boot bool) error {
	verbose("Intent to boot %s", i.path)

	distro, ok := supportedDistros[*distroName]
	if !ok {
		return fmt.Errorf("Could not infer ISO type based on filename.")
	}

	verbose("Using distro %s with boot config %s", *distroName, distro.BootConfig)

	var configs []Boot.OSImage
	if distro.BootConfig != "" {
		parsedConfigs, err := bootiso.ParseConfigFromISO(i.path, distro.BootConfig)
		if err != nil {
			return err
		}

		configs = append(configs, parsedConfigs...)
	}

	if len(distro.CustomConfigs) != 0 {
		customConfigs, err := bootiso.LoadCustomConfigs(i.path, distro.CustomConfigs)
		if err != nil {
			return err
		}

		configs = append(configs, customConfigs...)
	}

	if len(configs) == 0 {
		return fmt.Errorf("No valid configs were found.")
	}

	verbose("Get configs: %+v", configs)

	entries := []menu.Entry{}
	for _, config := range configs {
		entries = append(entries, &BootConfig{config})
	}

	config, ok := entries[0].(*BootConfig)
	if !ok {
		return fmt.Errorf("Could not convert selection to a boot image.")
	}

	paramTemplate, err := template.New("template").Parse(distro.KernelParams)
	if err != nil {
		return err
	}

	var kernelParams bytes.Buffer
	if err = paramTemplate.Execute(&kernelParams, cacheDev); err != nil {
		return err
	}

	if !boot {
		s := fmt.Sprintf("config.image %s, kernelparams.String() %s", config.image, kernelParams.String())
		return fmt.Errorf("Booting is disabled (see --dryrun flag), but otherwise would be [%s].", s)
	}
	err = bootiso.BootCachedISO(config.image, kernelParams.String()+" waitusb=10")

	// If kexec succeeds, we should not arrive here
	if err == nil {
		// TODO: We should know whether we tried using /sbin/kexec.
		err = fmt.Errorf("kexec failed, but gave no error. Consider trying kexec-tools.")
	}

	return err
}

// DownloadOption's exec lets user input the name of the iso they want
// if this iso is existed in the bookmark, use it's url
// elsewise ask for a download link
func (d *DownloadOption) exec() (menu.Entry, error) {

	link := supportedDistros[*distroName].Mirrors[0].Url

	filename := path.Base(link)

	// If the cachedir is not find, downloaded the iso to /tmp, else create a Downloaded dir in the cache dir.
	var fpath string
	var downloadDir string
	var err error

	downloadDir = os.TempDir()
	fpath = filepath.Join(downloadDir, filename)

	if err = download(link, fpath, downloadDir); err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("Download was canceled.")
		} else {
			return nil, err
		}
	}

	return &ISO{label: filename, path: fpath}, nil
}

// distroData downloads and parses the data in distros.json to a map[string]Distro.
func distroData() error {
	jsonPath := "/ci.json"

	// Parse the json file.
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("Could not read JSON file: %v\n", err)
	}

	err = json.Unmarshal([]byte(data), &supportedDistros)
	if err != nil {
		return fmt.Errorf("Could not unmarshal JSON file: %v\n", err)
	}

	return nil
}

func main() {
	flag.Parse()
	if *v {
		verbose = log.Printf
	}

	var err error
	var entry menu.Entry

	// get distro data
	err = distroData()
	if err != nil {
		log.Fatalf("Error on distroData(): %v", err.Error())
	}

	// DownloadOption
	entry = &DownloadOption{}
	entry, err = entry.(*DownloadOption).exec()
	if err != nil {
		log.Fatalf("Error on (*DownloadOption).exec(): %v", err.Error())
	}

	// ISO
	if err = entry.(*ISO).exec(!*dryRun); err != nil {
		log.Fatalf("Error on (*ISO).exec(): %v", err.Error())
	}
}
