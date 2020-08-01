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
			t.Errorf("Error msg are wrong, want %+v but get %+v", expected, err)
		}
	})

	t.Run("download_tinycore", func(t *testing.T) {
		fPath := "/tmp/test_tinycore.iso"
		url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
		if err := download(url, fPath); err != nil {
			t.Errorf("Fail to download: %+v", err)
		}
		if _, err := os.Stat(fPath); err != nil {
			t.Errorf("Fail to find downloaded file: %+v", err)
		}
		if err := os.Remove(fPath); err != nil {
			t.Errorf("Fail to remove test file: %+v", err)
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
				t.Errorf("Fail to execute downloadOption.exec(): %+v", err)
			}
			iso, ok := entry.(*ISO)
			if !ok {
				t.Errorf("Expected type *ISO, but get %T", entry)
			}
			if tt.want.label != iso.label || tt.want.path != iso.path {
				t.Errorf("Incorrect return. get %+v, want %+v", entry, tt.want)
			}
			if _, err := os.Stat(iso.path); err != nil {
				t.Errorf("Fail to find downloaded file: %+v", err)
			}

			if err := os.Remove(iso.path); err != nil {
				t.Errorf("Fail to remove test file: %+v", err)
			}
		})
	}

}

func TestISOOption(t *testing.T) {
	iso := &ISO{
		label: "TinyCorePure64.iso",
		path:  "./testdata/dirlevel1/dirlevel2/TinyCorePure64.iso",
	}

	if err := iso.exec(); err != nil {
		t.Errorf("Fail to execute iso.exec(): %+v", err)
	}

	if _, err := os.Stat(iso.path); err != nil {
		t.Errorf("Fail to find the iso file: %+v", err)
	}
}

func TestDirOption(t *testing.T) {
	// list of entries that would be hit in order in this test
	chosenEntries := []menu.Entry{
		&DirOption{
			label: "root dir",
			path:  "./testdata",
		},
		&DirOption{
			label: "dirlevel1",
			path:  "./testdata/dirlevel1",
		},
		&DirOption{
			label: "dirlevel2",
			path:  "./testdata/dirlevel1/dirlevel2",
		},
		&ISO{
			label: "TinyCorePure64.iso",
			path:  "./testdata/dirlevel1/dirlevel2/TinyCorePure64.iso",
		},
	}

	uiEvents := make(chan ui.Event)
	input := []string{"0", "<Enter>", "0", "<Enter>", "0", "<Enter>"}
	go pressKey(uiEvents, input)

	entry := chosenEntries[0]
	for i, e := range chosenEntries {
		if entry.Label() != e.Label() {
			t.Errorf("Get wrong chosen entry. get %+v, want %+v", entry, e)
		}
		// when i=3 the entry should be an ISO option which do no contains uiEvent as input
		var err error = nil
		if i < 3 {
			dirOption, ok := entry.(*DirOption)
			if !ok {
				t.Errorf("Expected type *DirOption, but get %T of entry %+v", entry, entry)
			}
			entry, err = dirOption.exec(uiEvents)
		} else {
			_, ok := entry.(*ISO)
			if !ok {
				t.Errorf("Expected type *ISO, but get %T of entry %+v", entry, entry)
			}
		}
		if err != nil {
			t.Errorf("Fail to execute option (%q)'s exec(): %+v", entry.Label(), err)
		}
	}

}
