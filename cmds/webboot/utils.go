package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

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
// todo: add a download progress bar
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
	if _, err := io.Copy(f, isoReader); err != nil {
		return fmt.Errorf("Fail to copy iso to a persistent memory device: %v", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("Fail to  close %s: %v", fPath, err)
	}
	verbose("%q is downloaded at %q\n", URL, fPath)
	return nil
}
