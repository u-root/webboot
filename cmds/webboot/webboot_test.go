package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	uiEvents := make(chan ui.Event)

	t.Run("error_link", func(t *testing.T) {
		errorLink := "errorlink"
		expected := fmt.Errorf("Get %q: unsupported protocol scheme \"\"", errorLink)
		if err := download(errorLink, "/tmp/test.iso", uiEvents); err.Error() != expected.Error() {
			t.Errorf("Expected %+v, received %+v", expected, err)
		}
	})

	t.Run("download_tinycore", func(t *testing.T) {
		fPath := "/tmp/test_tinycore.iso"
		url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
		if err := download(url, fPath, uiEvents); err != nil {
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
	tinycoreIso := &ISO{
		label: "TinyCorePure64-11.1.iso",
		path:  "testdata/Downloaded/TinyCorePure64-11.1.iso",
	}

	// Select custom distro, then type Tinycore URL manually
	customIndex := len(supportedDistros)
	tinycoreURL := supportedDistros["Tinycore"].url
	customCmd := []string{strconv.Itoa(customIndex), "<Enter>"}
	customCmd = append(customCmd, stringToKeypress(tinycoreURL)...)
	customCmd = append(customCmd, "<Enter>")

	for _, tt := range []struct {
		name  string
		input []string
		want  *ISO
	}{
		{
			name:  "test_bookmark",
			input: []string{strconv.Itoa(distroIndex("Tinycore")), "<Enter>"},
			want:  tinycoreIso,
		},
		{
			name:  "test_custom_url",
			input: customCmd,
			want:  tinycoreIso,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			go pressKey(uiEvents, tt.input)

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

func TestCancelDownload(t *testing.T) {
	uiEvents := make(chan ui.Event)
	keyPresses := []string{"0", "<Enter>", "<Escape>"}
	go pressKey(uiEvents, keyPresses)

	downloadOption := DownloadOption{}
	entry, err := downloadOption.exec(uiEvents, false, "./testdata")

	if _, ok := entry.(*BackOption); !ok {
		t.Errorf("Unknown return type %T!\n", entry)
	} else if err != nil {
		t.Errorf("Received error: %+v", err)
	}

	if err := os.RemoveAll("./testdata/Downloaded"); err != nil {
		t.Errorf("Fail to remove test file: %+v", err)
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
	input := []string{"0", "<Enter>", "<Escape>"}
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

func distroIndex(searchName string) int {
	var downloadOptions []string
	for distroName, _ := range supportedDistros {
		downloadOptions = append(downloadOptions, distroName)
	}
	sort.Strings(downloadOptions)

	for index, distroName := range downloadOptions {
		if distroName == searchName {
			return index
		}
	}
	return -1
}

func stringToKeypress(str string) []string {
	var keyPresses []string
	for i := 0; i < len(str); i++ {
		keyPresses = append(keyPresses, str[i:i+1])
	}
	return keyPresses
}
