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

func TestDownloadByLinkOption(t *testing.T) {
	url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	name := "test_download_by_link.iso"
	fPath := "/tmp/test_download_by_link.iso"

	uiEvents := make(chan ui.Event)
	input := append(strings.Split(url, ""), "<Enter>")
	input = append(input, strings.Split(name, "")...)
	input = append(input, "<Enter>")
	go pressKey(uiEvents, input)
	var entry menu.Entry = &DownloadByLink{uiEvents: uiEvents}

	if err := entry.Exec(); err != nil {
		t.Errorf("Fail to execute, error msg: %+v", err)
	}

	if _, err := os.Stat(fPath); err != nil {
		t.Errorf("Fail to find downloaded file, error msg: %+v", err)
	}

	if err := os.Remove(fPath); err != nil {
		t.Errorf("Fail to remove test file, error msg: %+v", err)
	}
}

func TestBookmarkISOOption(t *testing.T) {
	var entry menu.Entry = &BookMarkISO{
		url:   "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso",
		label: "TinyCorePure64-10.1.iso",
	}
	fPath := "/tmp/TinyCorePure64-10.1.iso"

	if err := entry.Exec(); err != nil {
		t.Errorf("Fail to execute, error msg: %+v", err)
	}

	if _, err := os.Stat(fPath); err != nil {
		t.Errorf("Fail to  find downloaded file, error msg: %+v", err)
	}

	if err := os.Remove(fPath); err != nil {
		t.Errorf("Fail to remove test file, error msg: %+v", err)
	}
}

func TestDownloadByBookmarkOption(t *testing.T) {
	uiEvents := make(chan ui.Event)
	input := []string{"0", "<Enter>", "0", "<Enter>", "0", "<Enter>"}
	go pressKey(uiEvents, input)
	fPath := "/tmp/TinyCorePure64-10.1.iso"
	var entry menu.Entry = &BookmarkGroup{uiEvents: uiEvents, entries: bookmarks, label: ""}

	if err := entry.Exec(); err != nil {
		t.Errorf("Fail to execute, error msg: %+v", err)
	}

	if _, err := os.Stat(fPath); err != nil {
		t.Errorf("Fail to find downloaded file, error msg: %+v", err)
	}

	if err := os.Remove(fPath); err != nil {
		t.Errorf("Fail to remove test file, error msg: %+v", err)
	}
}

func TestInstallCachedISOOption(t *testing.T) {
	for _, tt := range []struct {
		name      string
		userInput []string
	}{
		{
			name:      "hit first iso in the first dir",
			userInput: []string{"0", "<Enter>", "0", "<Enter>"},
		},
		{
			name:      "hit second iso in the second dir",
			userInput: []string{"1", "<Enter>", "1", "<Enter>"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			go pressKey(uiEvents, tt.userInput)

			var entry menu.Entry = &DirGroup{uiEvents: uiEvents, path: "./testdata/", label: ""}
			err := entry.Exec()
			if err != nil {
				t.Errorf("Error: %v", err)
			}

		})
	}

}

func TestMainMenu(t *testing.T) {
	url := "http://tinycorelinux.net/10.x/x86_64/release/TinyCorePure64-10.1.iso"
	name := "test_download_by_link.iso"
	downloadByLinkInput := []string{"2", "<Enter>"}
	downloadByLinkInput = append(downloadByLinkInput, strings.Split(url, "")...)
	downloadByLinkInput = append(downloadByLinkInput, "<Enter>")
	downloadByLinkInput = append(downloadByLinkInput, strings.Split(name, "")...)
	downloadByLinkInput = append(downloadByLinkInput, "<Enter>")

	for _, tt := range []struct {
		name      string
		userInput []string
		check     func()
	}{
		{
			name:      "hit first cached iso in the first group",
			userInput: []string{"0", "<Enter>", "0", "<Enter>", "0", "<Enter>"},
		},
		{
			name:      "hit first cached iso in the second group",
			userInput: []string{"0", "<Enter>", "1", "<Enter>", "0", "<Enter>"},
		},
		{
			name:      "hit the second bookmark under Tinycore/TinyCorePure64",
			userInput: []string{"1", "<Enter>", "0", "<Enter>", "0", "<Enter>", "1", "<Enter>"},
			check: func() {
				fPath := "/tmp/Webboot_Tinycorepure.iso"
				if _, err := os.Stat(fPath); err != nil {
					t.Errorf("Fail to find downloaded file, error msg: %+v", err)
				}
				if err := os.Remove(fPath); err != nil {
					t.Errorf("Fail to remove test file, error msg: %+v", err)
				}
			},
		},
		{
			name:      "download iso by link",
			userInput: downloadByLinkInput,
			check: func() {
				fPath := "/tmp/test_download_by_link.iso"
				if _, err := os.Stat(fPath); err != nil {
					t.Errorf("Fail to find downloaded file, error msg: %+v", err)
				}

				if err := os.Remove(fPath); err != nil {
					t.Errorf("Fail to remove test file, error msg: %+v", err)
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			go pressKey(uiEvents, tt.userInput)

			err := getMainMenu("./testdata/", uiEvents)
			if err != nil {
				t.Errorf("Error: %v", err)
			}
			if tt.check != nil {
				tt.check()
			}
		})
	}

}
