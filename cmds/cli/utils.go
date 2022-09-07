package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"

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

// QEMU testing uses serial output, so termui cannot be used. Instead,
// download percentage is logged when it's roughly a whole number
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.received += float64(n)
	percentage := 100 * (wc.received / wc.expected)

	// `percentage` is logged when "close enough" to a whole number, which depends
	// on how big the written chunk is (to account for different download
	// conditions causing chunks to vary in size)
	threshold := float64(n) / wc.expected * 100
	const megabyte = 1_000_000
	const gigabyte = 1_000_000_000
	if math.Abs(percentage-math.Trunc(percentage)) < threshold {
		verbose("Downloading... %.2f%% (%.3f MB / %.3f GB)",
			100*(wc.received/wc.expected),
			wc.received/megabyte,
			wc.expected/gigabyte)
	}

	return n, nil
}

func (wc *WriteCounter) Close() {
	wc.progress.Close()
}

// download() will download a file from URL and save it to a temp file
// If the download succeeds, the temp file will be copied to fPath
func download(URL, fPath, downloadDir string) error {
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
	if resp.StatusCode != 200 {
		return fmt.Errorf("Received http status code %s", resp.Status)
	}
	defer resp.Body.Close()

	tempFile, err := ioutil.TempFile(downloadDir, "iso-download-")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	counter := NewWriteCounter(resp.ContentLength)

	if _, err = io.Copy(tempFile, io.TeeReader(resp.Body, &counter)); err != nil {
		counter.Close()
		return err
	}

	counter.Close()

	if _, err = tempFile.Seek(0, 0); err != nil {
		return err
	}

	err = os.Rename(tempFile.Name(), fPath)
	if err != nil {
		return fmt.Errorf("Error on os.Rename: %v", err)
	}

	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
}

func supportedDistroEntries() []menu.Entry {
	entries := []menu.Entry{}
	for distroName := range supportedDistros {
		entries = append(entries, &Config{label: distroName})
	}

	sort.Slice(entries[:], func(i, j int) bool {
		return entries[i].Label() < entries[j].Label()
	})

	return entries
}
