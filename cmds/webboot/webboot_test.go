package main

import (
	"fmt"
	"os"
	"testing"
)

func TestDownload(t *testing.T) {
	t.Run("error_link", func(t *testing.T) {
		expected := fmt.Errorf("%q: linkopen only supports file://, https://, and http:// schemes", "errorlink")
		if err := download("errorlink", "/tmp", "test.iso"); err.Error() != expected.Error() {
			t.Errorf("Error: error msg are wrong, want %+v but get %+v", expected, err)
		}
	})
	t.Run("download_tinycore", func(t *testing.T) {
		if err := download("http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso", "/tmp", "test_tinycore.iso"); err != nil {
			t.Errorf("Error: when downloading, get error %+v", err)
		}
		fPath := "/tmp/test_tinycore.iso"
		if _, err := os.Stat(fPath); err != nil {
			t.Errorf("Error: do not find downloaded file, error msg: %+v", err)
		}
		if err := os.Remove(fPath); err != nil {
			t.Errorf("Error: can not remove test file, error msg: %+v", err)
		}
	})
}
