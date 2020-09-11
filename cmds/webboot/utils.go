package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"

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
	wc.progress.Update(fmt.Sprintf("Downloading... %.2f%% (%.3f MB)", 100*(wc.received/wc.expected), wc.received/1000000))
	return n, nil
}

func (wc *WriteCounter) Close() {
	wc.progress.Close()
}

func linkOpen(URL string) (io.ReadCloser, int64, error) {
	u, err := url.Parse(URL)
	if err != nil {
		log.Fatal(err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, -1, fmt.Errorf("%q: linkopen only supports http://, and https:// schemes", URL)
	}

	resp, err := http.Get(URL)
	if err != nil {
		return nil, -1, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, -1, fmt.Errorf("HTTP Get failed: %v", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

// download will download a file from URL and save it as fPath
func download(URL, fPath string) error {
	isoReader, isoSize, err := linkOpen(URL)
	if err != nil {
		return err
	}
	defer isoReader.Close()
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer f.Close()

	counter := NewWriteCounter(isoSize)
	if _, err = io.Copy(f, io.TeeReader(isoReader, &counter)); err != nil {
		return fmt.Errorf("Fail to copy iso to a persistent memory device: %v", err)
	}
	counter.Close()

	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
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
