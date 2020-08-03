package bootiso

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
)

func TestParseConfigFromISO(t *testing.T) {
	isoPath, err := getIsoPath()
	if err != nil {
		t.Error(err)
	}

	configOpts, err := ParseConfigFromISO(isoPath)
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

func downloadTestISO(isoPath string) error {
	resp, err := http.Get("https://github.com/u-root/webboot-distro/raw/master/iso/tinycore/10.x/x86_64/release/TinyCorePure64.iso")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Get failed: %v", resp.StatusCode)
	}

	file, err := os.Create(isoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func getIsoPath() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homedir + "/TinyCorePure64.iso", nil
}

func TestMain(m *testing.M) {
	// Need a test ISO at the home directory
	isoPath, err := getIsoPath()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stat(isoPath); err == nil {
		fmt.Println("ISO file was found.")
	} else if os.IsNotExist(err) {
		fmt.Print("No ISO found. Downloading...")
		err := downloadTestISO(isoPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print("DONE!\n")
	}

	os.Exit(m.Run())
}
