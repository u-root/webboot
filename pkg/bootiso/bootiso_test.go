package bootiso

import (
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
		checksumPath string
		checksumType string
		valid        bool
	}{
		{
			name:         "valid_md5",
			checksumPath: "testdata/TinyCorePure64.md5.txt",
			checksumType: "md5",
			valid:        true,
		},
		{
			name:         "valid_sha256",
			checksumPath: "testdata/TinyCorePure64.sha256.txt",
			checksumType: "sha256",
			valid:        true,
		},
		{
			name:         "invalid_md5",
			checksumPath: "testdata/TinyCorePure64.sha256.txt",
			checksumType: "md5",
			valid:        false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid, err := VerifyChecksum(isoPath, test.checksumPath, test.checksumType)
			if err != nil {
				t.Error(err)
			} else if valid != test.valid {
				t.Errorf("Checksum validation was expected to result in %t.\n", test.valid)
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
