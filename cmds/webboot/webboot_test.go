package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

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

func nextMenuReady(menus <-chan string) string {
	return <-menus
}

// The following files can be downloaded:
const (
	// randomISO is a file containing 1 mebibytes of random data.
	randomISO = "random1MiB.iso"
	// inititeISO is a file which is infinite bytes and takes forever to
	// download.
	infiniteISO = "infinite.iso"
)

// MiB is 1 mebibyte.
const MiB = 1024 * 1024

// TestMain is run once before all tests.
func TestMain(m *testing.M) {
	// Launch the fake ISO server.
	server, err := startFakeISOServer()
	if err != nil {
		log.Fatalf("error starting fake ISO server: %v", err)
	}
	defer server.stop()

	// Replace the supportedDistros list with a fake list for testing.
	supportedDistros = map[string]Distro{
		"FakeArch": {
			// This checksum corresponds to the random data for random1MiB.iso.
			Checksum:     "d6e467cd833bfabaefd652cdea1c7bd8318392f703ddf73160c324f515b965a3",
			ChecksumType: "sha256",
			Mirrors: []Mirror{
				{
					Name: "Default",
					Url:  server.url(randomISO),
				},
				{
					Name: "Arizona",
					Url:  server.url(randomISO),
				},
			},
		},
		"FakeTinycore": {
			// This checksum corresponds to the random data for random1MiB.iso.
			Checksum:     "d6e467cd833bfabaefd652cdea1c7bd8318392f703ddf73160c324f515b965a3",
			ChecksumType: "sha256",
			Mirrors: []Mirror{
				{
					Name: "Default",
					Url:  server.url(randomISO),
				},
			},
		},
		"InfiniteOS": {
			Mirrors: []Mirror{
				{Url: server.url(infiniteISO)},
			},
		},
	}

	// Run tests.
	os.Exit(m.Run())
}

// fakeISOServer serves fake ISO images for testing.
type fakeISOServer struct {
	server *http.Server
	port   int
}

// startFakeISOServers starts serving ISOs on 127.0.0.1. The port is in the
// returned struct.
func startFakeISOServer() (*fakeISOServer, error) {
	f := &fakeISOServer{}
	f.server = &http.Server{
		Handler: f,
	}

	// Find an unused port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("error creating free port: %v", err)
	}
	f.port = l.Addr().(*net.TCPAddr).Port

	go func() {
		if err := f.server.Serve(l); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return f, nil
}

// stop stops serving ISOs.
func (f fakeISOServer) stop() {
	f.server.Close()
}

// url returns the download url for the given filename.
func (f fakeISOServer) url(filename string) string {
	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", f.port),
		Path:   filename,
	}
	return u.String()
}

// ServeHTTP handles HTTP requests for the fakeISOServer.
func (f fakeISOServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Random number generator.
	rr := rand.New(rand.NewSource(99))

	switch r.URL.Path {
	case "/" + randomISO:
		w.WriteHeader(200)
		io.CopyN(w, rr, MiB)

	case "/" + infiniteISO:
		w.WriteHeader(200)

		ctx := r.Context()
		for {
			// Write 1 mebibyte every 1 second until the connection
			// is closed.
			io.CopyN(w, rr, MiB)
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Second):
			}
		}

	default:
		w.WriteHeader(404)
	}
}

func TestDownload(t *testing.T) {
	uiEvents := make(chan ui.Event)

	t.Run("error_link", func(t *testing.T) {
		errorLink := "errorlink"
		expected := fmt.Errorf("Get %q: unsupported protocol scheme \"\"", errorLink)
		if err := download(errorLink, "/tmp/test.iso", "/testdata", uiEvents); err.Error() != expected.Error() {
			t.Errorf("Expected %+v, received %+v", expected, err)
		}
	})

	t.Run("download_tinycore", func(t *testing.T) {
		// Create a temporary directory for the download.
		tmpDir, err := filepath.Abs(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		fPath := filepath.Join(tmpDir, "test_download.iso")

		// Download the ISO from the fake server.
		u := supportedDistros["FakeTinycore"].Mirrors[0].Url
		if err := download(u, fPath, tmpDir, uiEvents); err != nil {
			t.Fatalf("Fail to download: %+v", err)
		}
		s, err := os.Stat(fPath)
		if err != nil {
			t.Fatalf("Fail to find downloaded file: %+v", err)
		}
		if s.Size() != MiB {
			t.Fatalf("Expected download size of %d; got %d", MiB, s.Size())
		}

	})
}

func TestGetJsonLink(t *testing.T) {
	for _, tt := range []struct {
		name  string
		human func(chan ui.Event, <-chan string)
		want  string
	}{
		{
			name: "test_downloaded",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
			want: "https://raw.githubusercontent.com/u-root/webboot/main/cmds/webboot/distros.json",
		},
		{
			name: "test_local",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"1", "<Enter>"})
			},
			want: "./distros.json",
		},
		{
			name: "test_custom",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"2", "<Enter>"})
				nextMenuReady(menus)
				pressKey(uiEvents, stringToKeypress("https://raw.githubusercontent.com/u-root/webboot/main/cmds/webboot/distros.json"))
				pressKey(uiEvents, []string{"<Enter>"})
			},
			want: "https://raw.githubusercontent.com/u-root/webboot/main/cmds/webboot/distros.json",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			menus := make(chan string)
			go tt.human(uiEvents, menus)
			got, _, err := getJsonLink(uiEvents, menus)

			if err != nil {
				t.Errorf("Error in getJsonLink(): %v", err)
			} else if got != tt.want {
				t.Errorf("%s: Got %s but want %s", tt.name, got, tt.want)
			}
		})
	}

}

func TestDistroData(t *testing.T) {
	for _, tt := range []struct {
		name  string
		human func(chan ui.Event, <-chan string)
	}{
		{
			name: "test_good_link",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name: "test_bad_link",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				// User chooses to enter a custom link.
				pressKey(uiEvents, []string{"2", "<Enter>"})
				nextMenuReady(menus)
				// The link is valid but can't be downloaded.
				pressKey(uiEvents, stringToKeypress("https://raw.githubusercontent.com/u-root/webboot/main/cmds/webboot/fake_link.json"))
				pressKey(uiEvents, []string{"<Enter>"})
				nextMenuReady(menus)
				// User presses 0 to continue with local json file.
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			menus := make(chan string)
			go tt.human(uiEvents, menus)
			err := distroData(uiEvents, menus, "./testdata")
			if err != nil {
				t.Fatalf("Error on distroData: %v", err)
			}

			if len(supportedDistros) == 0 {
				t.Fatalf("Got empty distro list, want provided JSON file to be unmarshaled into supportedDistros.")
			}

			for distroName := range supportedDistros {
				if len(supportedDistros[distroName].Mirrors) == 0 {
					t.Fatalf("Got empty mirror list in %s, want provided JSON file to be unmarshaled into supportedDistros.", distroName)
				}
			}
		})
	}
}

func TestDownloadOption(t *testing.T) {
	tinycoreIso := &ISO{
		label: randomISO,
		path:  filepath.Join("testdata/Downloaded", randomISO),
	}

	// Select custom distro, then type Tinycore URL manually
	customIndex := len(supportedDistros)
	tinycoreURL := supportedDistros["FakeTinycore"].Mirrors[0].Url
	tinycoreIndex, err := distroIndex("FakeTinycore")
	if err != nil {
		t.Fatalf("Error on distroIndex: %v", err)
	}
	for _, tt := range []struct {
		name  string
		want  *ISO
		human func(chan ui.Event, <-chan string)
	}{
		{
			name: "test_bookmark",
			want: tinycoreIso,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				// Distros selection menu
				pressKey(uiEvents, []string{strconv.Itoa(tinycoreIndex), "<Enter>"})
				nextMenuReady(menus)
				// Mirrors selection menu
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name: "test_custom_url",
			want: tinycoreIso,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{strconv.Itoa(customIndex), "<Enter>"})
				nextMenuReady(menus)
				pressKey(uiEvents, stringToKeypress(tinycoreURL))
				pressKey(uiEvents, []string{"<Enter>"})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			menus := make(chan string)
			go tt.human(uiEvents, menus)
			downloadOption := DownloadOption{}
			entry, err := downloadOption.exec(uiEvents, menus, false, "./testdata")

			if err != nil {
				t.Fatalf("Fail to execute downloadOption.exec(): %+v", err)
			}
			iso, ok := entry.(*ISO)
			if !ok {
				t.Fatalf("Expected type *ISO, but get %T", entry)
			}
			if tt.want.label != iso.label || tt.want.path != iso.path {
				t.Fatalf("Incorrect return. get %+v, want %+v", entry, tt.want)
			}
			if _, err := os.Stat(iso.path); err != nil {
				t.Fatalf("Fail to find downloaded file: %+v", err)
			}
			if err := os.RemoveAll("./testdata/Downloaded"); err != nil {
				t.Fatalf("Fail to remove test file: %+v", err)
			}
		})
	}
}

func TestCancelDownload(t *testing.T) {
	uiEvents := make(chan ui.Event)
	menus := make(chan string)
	index, err := distroIndex("InfiniteOS")
	if err != nil {
		t.Fatalf("Error on distroIndex: %v", err)
	}

	// InfiniteOS will take forever to download and must be cancelled.
	go func() {
		nextMenuReady(menus)
		// Distros selection menu
		pressKey(uiEvents, []string{strconv.Itoa(index), "<Enter>"})
		nextMenuReady(menus)
		// Mirrors selection menu
		pressKey(uiEvents, []string{"0", "<Enter>", "<Escape>"})
	}()

	downloadOption := DownloadOption{}
	_, err = downloadOption.exec(uiEvents, menus, false, "./testdata")

	if err == nil {
		t.Errorf("Got nil error; expected 'Download was canceled.'")
	}

	if err != nil && err.Error() != "Download was canceled." {
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
	menus := make(chan string)

	var entry menu.Entry = &DirOption{label: "root dir", path: "./testdata"}
	var err error = nil
	for {
		if dirOption, ok := entry.(*DirOption); ok {
			go func() {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			}()
			entry, err = dirOption.exec(uiEvents, menus)
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

// TestBackOption tests behavior when the escape key is pressed on a menu.
func TestBackOption(t *testing.T) {

	uiEvents := make(chan ui.Event)
	menus := make(chan string)

	var entry menu.Entry = &DirOption{path: "./testdata"}
	var err error = nil

	go func() {
		for i := 0; i < 2; i++ {
			nextMenuReady(menus)
			pressKey(uiEvents, []string{"0", "<Enter>"})
			nextMenuReady(menus)
			pressKey(uiEvents, []string{"<Escape>"})
		}
	}()
	// The first loop tests ./testdata - a back request should not have any effect
	// because we are already at the first possible menu.
	// The second loop tests testdata/dirlevel1 - a back request should return the previous menu.
	for i := 0; i < 2; i++ {
		if dirOption, ok := entry.(*DirOption); ok {
			currentPath := dirOption.path
			entry, err = dirOption.exec(uiEvents, menus)
			if err != nil && err != menu.BackRequest {
				t.Fatalf("Fail to execute option (%q)'s exec(): %+v", entry.Label(), err)
			} else if err == menu.BackRequest {
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
			t.Fatalf("Get incorrect dir option, want \"testdata\", get %s", dirOption.path)
		}
	}
}

func TestDisplayChecksumPrompt(t *testing.T) {
	// test data
	var testDistros = map[string]Distro{
		"FakeDistro": {
			Checksum:     "1234567",
			ChecksumType: "sha256",
		},
		"FakeDistroNoChecksum": {},
		"FakeDistroGoodChecksum": {
			Checksum:     "407dc87b95afbe268e760313971041860f36e953a2116db03418a98ce46d61bc",
			ChecksumType: "sha256",
		},
	}

	type test struct {
		name       string
		distroName string
		want       string
		human      func(chan ui.Event, <-chan string)
	}

	tests := []test{
		{
			name:       "Incorrect checksum, don't proceed",
			distroName: "FakeDistro",
			want:       "*main.DownloadOption",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"1", "<Enter>"})
			},
		},
		{
			name:       "Incorrect checksum, proceed",
			distroName: "FakeDistro",
			want:       "<nil>",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name:       "No checksum, don't proceed",
			distroName: "FakeDistroNoChecksum",
			want:       "*main.DownloadOption",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"1", "<Enter>"})
			},
		},
		{
			name:       "No checksum, proceed",
			distroName: "FakeDistroNoChecksum",
			want:       "<nil>",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name:       "Correct checksum",
			distroName: "FakeDistroGoodChecksum",
			want:       "<nil>",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{})
			},
		},
	}

	for _, tc := range tests {
		uiEvents := make(chan ui.Event)
		menus := make(chan string)

		t.Run(tc.name, func(t *testing.T) {
			go tc.human(uiEvents, menus)
			menu, err := displayChecksumPrompt(uiEvents, menus, testDistros, tc.distroName, "testdata/dirlevel1/fakeDistro.iso")
			if err != nil {
				t.Errorf("Error on displayChecksumPrompt: %v", err)
			} else if got := fmt.Sprintf("%T", menu); got != tc.want {
				t.Errorf("%s: Got %s but want %s", tc.name, got, tc.want)
			}
		})
	}
}

func distroIndex(searchName string) (int, error) {
	var downloadOptions []string
	for distroName := range supportedDistros {
		downloadOptions = append(downloadOptions, distroName)
	}
	sort.Strings(downloadOptions)

	var distroList string

	for index, distroName := range downloadOptions {
		distroList += fmt.Sprintf("%s ", distroName)
		if distroName == searchName {
			return index, nil
		}
	}

	return -1, fmt.Errorf("could not find distro %s. Here are the available distros: %s", searchName, distroList)
}

func stringToKeypress(str string) []string {
	var keyPresses []string
	for i := 0; i < len(str); i++ {
		keyPresses = append(keyPresses, str[i:i+1])
	}
	return keyPresses
}

func TestDefaultMirrorNameAndLinkCheck(t *testing.T) {
	uiEvents := make(chan ui.Event)
	menus := make(chan string)

	go func() {
		nextMenuReady(menus)
		pressKey(uiEvents, []string{"0", "<Enter>"})
	}()

	entry := &Config{label: "FakeArch"}
	u, m, err := mirrorMenu(entry, uiEvents, menus, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tl := supportedDistros["FakeArch"].Mirrors[0].Url
	if u != tl {
		t.Fatalf("Wrong mirror link. Got %q, want %q", u, tl)
	}
	if m != "Default" {
		t.Fatalf("Wrong mirror name. Got %q, want %q", m, "Default")
	}
}

func TestMirrorNameAndLinkCheck(t *testing.T) {
	uiEvents := make(chan ui.Event)
	menus := make(chan string)

	go func() {
		nextMenuReady(menus)
		pressKey(uiEvents, []string{"1", "<Enter>"})
	}()

	entry := &Config{label: "FakeArch"}
	u, m, err := mirrorMenu(entry, uiEvents, menus, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	tl := supportedDistros["FakeArch"].Mirrors[0].Url
	if u != tl {
		t.Fatalf("Wrong mirror link. Got %q, want %q", u, tl)
	}
	if m != "Arizona" {
		t.Fatalf("Wrong mirror name. Got %q, want %q", m, "Arizona")
	}
}

func TestMirrorNameAndLinkCheckBad(t *testing.T) {
	t.Skip("TODO: This test is disabled until the menu package is fixed.")
	uiEvents := make(chan ui.Event)
	menus := make(chan string)

	go func() {
		nextMenuReady(menus)
		pressKey(uiEvents, []string{"9", "<Enter>"})
	}()

	entry := &Config{label: "FakeArch"}
	_, _, err := mirrorMenu(entry, uiEvents, menus, "")
	if err == nil {
		t.Fatalf("Bad mirror selection: got nil, want error")
	}
}
