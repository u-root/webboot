package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

func pressKey(ch chan ui.Event, input []string) {
	var key ui.Event
	for _, id := range input {
		key = ui.Event{
			Type: ui.KeyboardEvent,
			ID:   id,
		}
		ch <- key
	}
}

func TestDownload(t *testing.T) {
	t.Run("error_link", func(t *testing.T) {
		expected := fmt.Errorf("%q: linkopen only supports file://, https://, and http:// schemes", "errorlink")
		if err := download("errorlink", "/tmp/test.iso"); err.Error() != expected.Error() {
			t.Errorf("Error: error msg are wrong, want %+v but get %+v", expected, err)
		}
	})

	t.Run("download_tinycore", func(t *testing.T) {
		fPath := "/tmp/test_tinycore.iso"
		url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
		if err := download(url, fPath); err != nil {
			t.Errorf("Error: when downloading, get error %+v", err)
		}
		if _, err := os.Stat(fPath); err != nil {
			t.Errorf("Error: do not find downloaded file, error msg: %+v", err)
		}
		if err := os.Remove(fPath); err != nil {
			t.Errorf("Error: can not remove test file, error msg: %+v", err)
		}
	})
}

func TestDownloadByLink(t *testing.T) {
	var entry menu.Entry = &DownloadByLink{}
	url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	isoName := "test_download_by_link.iso"
	fPath := "/tmp/test_download_by_link.iso"

	uiEvents := make(chan ui.Event)
	input := append(strings.Split(url, ""), "<Enter>")
	input = append(input, strings.Split(isoName, "")...)
	input = append(input, "<Enter>")
	go pressKey(uiEvents, input)

	if err := entry.Exec(uiEvents); err != nil {
		t.Errorf("Error: Fail to execute, error msg: %+v", err)
	}

	if _, err := os.Stat(fPath); err != nil {
		t.Errorf("Error: do not find downloaded file, error msg: %+v", err)
	}

	if err := os.Remove(fPath); err != nil {
		t.Errorf("Error: can not remove test file, error msg: %+v", err)
	}
}
