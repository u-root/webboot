package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	ui "github.com/gizak/termui/v3"
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
			t.Errorf("Error msg are wrong, want %+v but get %+v", expected, err)
		}
	})

	t.Run("download_tinycore", func(t *testing.T) {
		fPath := "/tmp/test_tinycore.iso"
		url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
		if err := download(url, fPath); err != nil {
			t.Errorf("Fail to download, get error %+v", err)
		}
		if _, err := os.Stat(fPath); err != nil {
			t.Errorf("Fail to find downloaded file, error msg: %+v", err)
		}
		if err := os.Remove(fPath); err != nil {
			t.Errorf("Fail to remove test file, error msg: %+v", err)
		}
	})
}

func TestDownloadOption(t *testing.T) {
	bookmarkIso := &ISO{
		label: "TinyCorePure64-10.1.iso",
		path:  "/tmp/TinyCorePure64-10.1.iso",
	}
	downloadByLinkIso := &ISO{
		label: "test_download_by_link.iso",
		path:  "/tmp/test_download_by_link.iso",
	}
	downloadLink := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"

	for _, tt := range []struct {
		name  string
		label []string
		url   []string
		want  *ISO
	}{
		{
			name:  "test_bookmark",
			label: append(strings.Split(bookmarkIso.label, ""), "<Enter>"),
			want:  bookmarkIso,
		},
		{
			name:  "test_download_by_link",
			label: append(strings.Split(downloadByLinkIso.label, ""), "<Enter>"),
			url:   append(strings.Split(downloadLink, ""), "<Enter>"),
			want:  downloadByLinkIso,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			input := append(tt.label, tt.url...)
			go pressKey(uiEvents, input)

			downloadOption := DownloadOption{}
			entry, err := downloadOption.exec(uiEvents)

			if err != nil {
				t.Errorf("Error: %v", err)
			}
			iso, ok := entry.(*ISO)
			if !ok {
				t.Errorf("Fail to transfor the entry to *ISO, error msg: %+v", err)
			}
			if tt.want.label != iso.label || tt.want.path != iso.path {
				t.Errorf("Incorrect return. get %+v, want %+v", entry, tt.want)
			}
			if _, err := os.Stat(iso.path); err != nil {
				t.Errorf("Fail to find downloaded file, error msg: %+v", err)
			}

			if err := os.Remove(iso.path); err != nil {
				t.Errorf("Fail to remove test file, error msg: %+v", err)
			}
		})
	}

}

func TestISOOption(t *testing.T) {
	iso := &ISO{
		label: "TinyCorePure64-10.1.iso",
		path:  "./testdata/Tinycore/TinyCorePure64-10.1.iso",
	}

	if err := iso.exec(); err != nil {
		t.Errorf("Fail to execute, error msg: %+v", err)
	}

	if _, err := os.Stat(iso.path); err != nil {
		t.Errorf("Fail to find the iso file, error msg: %+v", err)
	}
}
