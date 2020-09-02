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

func TestMain(m *testing.M) {
	if _, err := os.Stat(isoPath); err != nil {
		log.Fatal("ISO file was not found in the testdata directory.")
	}

	os.Exit(m.Run())
}
