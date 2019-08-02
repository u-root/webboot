package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type test struct {
	name         string
	linkOrName   string
	md5link      string
	expectedLink string
	expectedName string
}

func TestParseArg(t *testing.T) {
	tests := []test{
		{name: "Debian", linkOrName: "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.0.0-amd64-netinst.iso", expectedLink: "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-10.0.0-amd64-netinst.iso", expectedName: "debian-10.0.0-amd64-netinst.iso"},
		{name: "TinyCore", linkOrName: "tinycore", expectedLink: "http://tinycorelinux.net/10.x/x86_64/release/CorePure64-10.1.iso", expectedName: "tinycore"},
		{name: "Ubuntu", linkOrName: "http://releases.ubuntu.com/18.04.2/ubuntu-18.04.2-desktop-amd64.iso", expectedLink: "http://releases.ubuntu.com/18.04.2/ubuntu-18.04.2-desktop-amd64.iso", expectedName: "ubuntu-18.04.2-desktop-amd64.iso"},
	}

	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {
			link, filename, err := parseArg(v.linkOrName)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Compare(link, v.expectedLink) != 0 {
				t.Errorf("Looking up URL for %v: got %v, want %v", v.linkOrName, link, v.expectedLink)
			}
			if strings.Compare(filename, v.expectedName) != 0 {
				t.Errorf("Looking up the file name for %v: got %v, want %v", v.linkOrName, filename, v.expectedName)
			}
		})
	}
}

func TestName(t *testing.T) {
	tests := []test{
		{name: "TinyCore", linkOrName: "http://tinycorelinux.net/10.x/x86_64/release/CorePure64-10.1.iso", expectedName: "CorePure64-10.1.iso"},
		{name: "ArchLinux", linkOrName: "http://mirrors.edge.kernel.org/archlinux/iso/2019.08.01/archlinux-2019.08.01-x86_64.iso", expectedName: "archlinux-2019.08.01-x86_64.iso"},
		{name: "Ubuntu", linkOrName: "http://releases.ubuntu.com/18.04.2/ubuntu-18.04.2-desktop-amd64.iso", expectedName: "ubuntu-18.04.2-desktop-amd64.iso"},
		{name: "HtmlFile", linkOrName: "http://releases.ubuntu.com", expectedName: "index.html"},
	}
	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {
			filename, err := name(v.linkOrName)
			if err != nil {
				t.Fatalf("Looking up the file name for %v: got %v, want nil", v.linkOrName, err)
			}
			if strings.Compare(filename, v.expectedName) != 0 {
				t.Errorf("Looking for name to generate for %v: got %v, want %v", v.linkOrName, filename, v.expectedName)
			}
		})
	}

}

func TestWrite(t *testing.T) {
	content := []byte("temporary file's content")
	tmpDir, err := ioutil.TempDir("", "tmpDir")
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	tmpfile := filepath.Join(tmpDir, "tmpfile")
	if err := ioutil.WriteFile(tmpfile, content, 0600); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(tmpfile)
	if err != nil {
		t.Fatalf("Failed to open %v: got %v, want nil", tmpfile, err)
	}

	testfn := filepath.Join(tmpDir, "testfile")
	err = write(file, testfn)

	read, err := ioutil.ReadFile(testfn)
	if !bytes.Equal(content, read) {
		t.Fatalf("Failed to write %v: got %v, want %v", testfn, string(read), string(content))
	}
}

func TestLinkOpen(t *testing.T) {
	tests := []test{
		{name: "TinyCore", linkOrName: "http://tinycorelinux.net/10.x/x86_64/release/CorePure64-10.1.iso", md5link: "http://tinycorelinux.net/10.x/x86_64/release/CorePure64-10.1.iso.md5.txt"},
		{name: "ArchLinux", linkOrName: "http://mirrors.edge.kernel.org/archlinux/iso/2019.08.01/archlinux-2019.08.01-x86_64.iso", md5link: "http://mirrors.edge.kernel.org/archlinux/iso/2019.08.01/md5sums.txt"},
		//		Ubuntu takes too long to download right now for testing purposes. Test fails due to the download and testing being too long.
		//		{name: "Ubuntu", linkOrName: "http://old-releases.ubuntu.com/releases/18.04.2/ubuntu-18.04.2-desktop-amd64.iso", md5link: "http://old-releases.ubuntu.com/releases/18.04.2/MD5SUMS"},
	}

	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {
			r, err := linkOpen(v.linkOrName)
			if err != nil {
				t.Fatalf("linkOpen %v: got %v, want nil", v.linkOrName, err)
			}

			body, err := ioutil.ReadAll(r)
			if err != nil {
				t.Fatalf("Failed to read %v: got %v, want nil", v.linkOrName, err)
			}
			//computes the hash of the ISO & converts it into a string to compare with md5.txt hash
			md := md5.Sum(body)
			hash := hex.EncodeToString(md[:])

			c, err := linkOpen(v.md5link)
			if err != nil {
				t.Fatalf("linkOpen md5 %v: got %v, want nil", v.md5link, err)
			}

			var bytes []byte
			if bytes, err = ioutil.ReadAll(c); err != nil {
				t.Fatalf("Failed to read md5 %v: got %v, want nil", v.md5link, err)
			}
			// md5sum files contain a list of hashes. Here, we check each one
			var found bool
			for _, line := range strings.Split(string(bytes), "\n") {
				if f := strings.Fields(line); len(f) != 0 && f[0] == hash {
					found = true
				}
			}

			if found != true {
				t.Fatalf("Checking md5sum of %v against hashes in %v: could not find a hash to match %v", v.linkOrName, v.md5link, hash)
			}
		})
	}
}
