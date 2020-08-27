package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/u-root/webboot/pkg/menu"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer
type WriteCounter struct {
	total    float64
	progress menu.Progress
}

func NewWriteCounter() WriteCounter {
	return WriteCounter{0, menu.NewProgress("", false)}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += float64(n)
	// print how many bytes have been writen to the file
	wc.progress.Update(fmt.Sprintf("\rDownloading... %.3f MB complete", wc.total/1000000))
	return n, nil
}

func (wc *WriteCounter) Close() {
	wc.progress.Close()
}

func linkOpen(URL string) (io.ReadCloser, error) {
	u, err := url.Parse(URL)
	if err != nil {
		log.Fatal(err)
	}
	switch u.Scheme {
	case "file":
		return os.Open(URL[7:])
	case "http", "https":
		resp, err := http.Get(URL)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP Get failed: %v", resp.StatusCode)
		}
		return resp.Body, nil
	}
	return nil, fmt.Errorf("%q: linkopen only supports file://, https://, and http:// schemes", URL)
}

// download will download a file from URL and save it as fPath
func download(URL, fPath string) error {
	isoReader, err := linkOpen(URL)
	if err != nil {
		return err
	}
	defer isoReader.Close()
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer f.Close()

	counter := NewWriteCounter()
	if _, err = io.Copy(f, io.TeeReader(isoReader, &counter)); err != nil {
		return fmt.Errorf("Fail to copy iso to a persistent memory device: %v", err)
	}
	counter.Close()

	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
}
