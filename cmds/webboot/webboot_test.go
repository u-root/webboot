package main

import (
	"fmt"
	"os"
	"path/filepath"
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
			t.Fatalf("Fail to download: %+v", err)
		}
		if _, err := os.Stat(fPath); err != nil {
			t.Fatalf("Fail to find downloaded file: %+v", err)
		}
		if err := os.Remove(fPath); err != nil {
			t.Fatalf("Fail to remove test file: %+v", err)
		}
	})
}

func TestDownloadOption(t *testing.T) {
	bookmarkIso := &ISO{
		label: "TinyCorePure64-10.1.iso",
		path:  "testdata/Downloaded/TinyCorePure64-10.1.iso",
	}
	downloadByLinkIso := &ISO{
		label: "test_download_by_link.iso",
		path:  "testdata/Downloaded/test_download_by_link.iso",
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
			entry, err := downloadOption.exec(uiEvents, false, "./testdata")

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
			if err := os.RemoveAll("./testdata/Downloaded"); err != nil {
				t.Errorf("Fail to remove test file: %+v", err)
			}
		})
	}

}

func TestDirOption(t *testing.T) {
	wanted := &ISO{
		label: "TinyCorePure64.iso",
		path:  "testdata/dirlevel1/dirlevel2/TinyCorePure64.iso",
	}

	uiEvents := make(chan ui.Event)
	input := []string{"0", "<Enter>", "0", "<Enter>", "0", "<Enter>"}
	go pressKey(uiEvents, input)

	var entry menu.Entry = &DirOption{label: "root dir", path: "./testdata"}
	var err error = nil
	for {
		if dirOption, ok := entry.(*DirOption); ok {
			entry, err = dirOption.exec(uiEvents)
			if err != nil {
				t.Fatalf("Fail to execute option (%q)'s exec(): %+v", entry.Label(), err)
			}
		} else if iso, ok := entry.(*ISO); ok {
			if iso.label != wanted.label || iso.path != wanted.path {
				t.Fatalf("Get wrong chosen iso. get %+v, want %+v", iso, wanted)
			}
			break
		} else {
			t.Fatalf("Unknown type. got entry %+v of type %T, wanted DirOption or ISO", entry, entry)
		}
	}
}

func TestBackOption(t *testing.T) {
	uiEvents := make(chan ui.Event)
	input := []string{"0", "<Enter>", "1", "<Enter>"}
	go pressKey(uiEvents, input)

	var entry menu.Entry = &DirOption{path: "./testdata"}
	var err error = nil
	for i := 0; i < 2; i++ {
		if dirOption, ok := entry.(*DirOption); ok {
			currentPath := dirOption.path
			entry, err = dirOption.exec(uiEvents)
			if err != nil {
				t.Fatalf("Fail to execute option (%q)'s exec(): %+v", entry.Label(), err)
			}
			if _, ok := entry.(*BackOption); ok {
				backTo := filepath.Dir(currentPath)
				entry = &DirOption{path: backTo}
			}
		} else {
			t.Fatalf("Unknown type. got entry %+v of type %T, wanted DirOption", entry, entry)
		}

	}
	if dirOption, ok := entry.(*DirOption); !ok {
		t.Fatalf("Incorrect result, want a DirOption, get %T", entry)
	} else {
		if dirOption.path != "testdata" {
			t.Fatalf("Get incorrect dir option, want \"datatest\", get %s", dirOption.path)
		}
	}
}

func TestBackOptionInDownload(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input []string
	}{
		{
			name:  "go_back_from_input_iso_name",
			input: []string{"<Escape>"},
		},
		{
			name:  "go_back_from_input_url",
			input: []string{"a", ".", "i", "s", "o", "<Enter>", "<Escape>"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			go pressKey(uiEvents, tt.input)

			d := &DownloadOption{}
			if entry, err := d.exec(uiEvents, false, ""); err != nil {
				t.Fatalf("Fail to execute download options's exec(): %+v", err)
			} else {
				if _, ok := entry.(*BackOption); !ok {
					t.Fatalf("Get incorrect return, want a BackOprion, get %T", entry)
				}
			}
		})
	}
}
