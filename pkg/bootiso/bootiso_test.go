package bootiso

import (
	"fmt"
	"log"
	"os"
	"testing"
)

var isoPath string = "testdata/TinyCorePure64.iso"

func TestParseConfigFromISO(t *testing.T) {
	configOpts, err := ParseConfigFromISO(isoPath, "syslinux")
	if err != nil {
		t.Error(err)
	}

	expectedLabels := [4]string{
		"Boot TinyCorePure64",
		"Boot TinyCorePure64 (on slow devices, waitusb=5)",
		"Boot Core (command line only).",
		"Boot Core (command line only on slow devices, waitusb=5)",
	}

	for i, config := range configOpts {
		if config.Label() != expectedLabels[i] {
			t.Error("Invalid configuration option found.")
		}
	}
}

func TestChecksum(t *testing.T) {
	for _, test := range []struct {
		name         string
		checksum     string
		checksumType string
		valid        bool
	}{
		{
			name:         "valid_md5",
			checksum:     "10a79ba7558598574cd396e7b1b057b7",
			checksumType: "md5",
			valid:        true,
		},
		{
			name:         "valid_sha256",
			checksum:     "01ce6b5f4e4f7e98eddc343fc14f1436fb1b0452e6b9f7e07461b6a089a909c1", 
			checksumType: "sha256",
			valid:        true,
		},
		{
			name:         "invalid_md5",
			checksum: "99979ba7558598574cd396e7b1b057b7",
			checksumType: "md5",
			valid:        false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid, err := VerifyChecksum(isoPath, test.checksum, test.checksumType)
			if err != nil {
				t.Error(err)
			} else if valid != test.valid {
				t.Errorf("Checksum validation was expected to result in %t.\n", test.valid)
			}
		})
	}
}

func TestCustomConfigs(t *testing.T) {
	var configs []Config
	for i := 0; i < 5; i++ {
		configs = append(configs, Config{
			Label:      "Custom Config " + fmt.Sprint(i),
			KernelPath: "/boot/vmlinuz64",
			InitrdPath: "/boot/corepure64.gz",
			Cmdline:    "loglevel=3 vga=791",
		})
	}

	for _, test := range []struct {
		name    string
		configs []Config
	}{
		{
			name:    "empty_list",
			configs: []Config{},
		},
		{
			name:    "single_config",
			configs: configs[:1],
		},
		{
			name:    "multiple_configs",
			configs: configs,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			images, err := LoadCustomConfigs(isoPath, test.configs)
			if err != nil {
				t.Error(err)
			} else if len(test.configs) != len(images) {
				t.Errorf("Test contained %d configs, but only received %d images.", len(test.configs), len(images))
			}

			for index, image := range images {
				if test.configs[index].Label != image.Label() {
					t.Errorf("Expected label %q but received %q.", test.configs[index].Label, image.Label())
				}
			}
		})
	}
}

func TestMain(m *testing.M) {
	if _, err := os.Stat(isoPath); err != nil {
		log.Fatal("ISO file was not found in the testdata directory.")
	}

	os.Exit(m.Run())
}
