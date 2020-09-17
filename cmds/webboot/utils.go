package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"

	ui "github.com/gizak/termui/v3"
	"github.com/u-root/webboot/pkg/menu"
)

// WriteCounter counts the number of bytes written to it. It implements an io.Writer
type WriteCounter struct {
	received float64
	expected float64
	progress menu.Progress
}

func NewWriteCounter(expectedSize int64) WriteCounter {
	return WriteCounter{0, float64(expectedSize), menu.NewProgress("", false)}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.received += float64(n)
	wc.progress.Update(fmt.Sprintf("Downloading... %.2f%% (%.3f MB)\n\nPress <Esc> to cancel.", 100*(wc.received/wc.expected), wc.received/1000000))
	return n, nil
}

func (wc *WriteCounter) Close() {
	wc.progress.Close()
}

// download will download a file from URL and save it as fPath
func download(URL, fPath string, uiEvents <-chan ui.Event) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer f.Close()

	go listenForCancel(ctx, cancel, uiEvents)
	counter := NewWriteCounter(resp.ContentLength)
	defer counter.Close()

	if _, err = io.Copy(f, io.TeeReader(resp.Body, &counter)); err != nil {
		return err
	}

	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
}

func listenForCancel(ctx context.Context, cancel context.CancelFunc, uiEvents <-chan ui.Event) {
	for {
		select {
		case k := <-uiEvents:
			if k.ID == "<Escape>" {
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func inferIsoType(isoName string) string {
	for distroName, distroInfo := range supportedDistros {
		match, _ := regexp.MatchString(distroInfo.isoPattern, isoName)
		if match {
			return distroName
		}
	}
	return ""
}

func supportedDistroEntries() []menu.Entry {
	entries := []menu.Entry{}
	for distroName, _ := range supportedDistros {
		entries = append(entries, &Config{label: distroName})
	}

	sort.Slice(entries[:], func(i, j int) bool {
		return entries[i].Label() < entries[j].Label()
	})

	return entries
}

func validURL(url string) (string, string, bool) {
	match, _ := regexp.MatchString("^https*://.+\\.iso$", url)
	if match {
		return url, "", true
	} else {
		return url, "Invalid URL.", false
	}
}
