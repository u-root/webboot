package menu

import (
	"strconv"
	"strings"
	"testing"

	ui "github.com/gizak/termui/v3"
)

type testEntry struct {
	message   string
	label     string
	isDefault bool
}

func (u *testEntry) Label() string {
	return u.label
}

func (u *testEntry) IsDefault() bool {
	return u.isDefault
}

func TestNewParagraph(t *testing.T) {
	testText := "newParagraph test"
	p := newParagraph(testText, false, 0, 50, 3)
	if testText != p.Text {
		t.Errorf("Incorrect value for p.Text. got: %v, want: %v", p.Text, testText)
	}
}

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

func TestProcessInputSimple(t *testing.T) {
	testText := "test"
	uiEvents := make(chan ui.Event)
	go pressKey(uiEvents, []string{"t", "e", "s", "t", "<Enter>"})

	input, _, err := processInput("test processInput simple", 0, 50, 1, AlwaysValid, uiEvents)

	if err != nil {
		t.Errorf("ProcessInput failed: %v", err)
	}
	if input != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", input, testText)
	}
}

func TestProcessInputComplex(t *testing.T) {
	testText := "100"
	uiEvents := make(chan ui.Event)
	// mock user input:
	// first input is bad input "bad"
	// second input is bad input "100a"
	// third input is good input "100" but contain a process of typo then backspace
	// now the warning text should be ""
	go pressKey(uiEvents, []string{"b", "a", "d", "<Enter>",
		"1", "0", "0", "a", "<Enter>",
		"1", "0", "a", "<Backspace>", "0", "<Enter>"})

	isValid := func(input string) (string, string, bool) {
		if _, err := strconv.ParseUint(input, 10, 32); err != nil {
			return "", "Input is not a valid entry number.", false
		}
		return input, "", true
	}

	input, _, err := processInput("test processInput complex", 0, 50, 1, isValid, uiEvents)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if input != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", input, testText)
	}
}

func TestProcessInputLong(t *testing.T) {
	uiEvents := make(chan ui.Event)
	testText := "splash=silent quiet  root=live:CDLABEL=openSUSE_Leap_15.2_KDE_Live " +
		"rd.live.image rd.live.overlay.persistent rd.live.overlay.cowfs=ext4" +
		"iso-scan/filename=openSUSE-Leap-15.2-KDE-Live-x86_64-Build31.135-Media.iso"

	var keyPresses []string
	for i := 1; i <= len(testText); i++ {
		keyPresses = append(keyPresses, testText[i-1:i])
	}
	keyPresses = append(keyPresses, "<Enter>")
	go pressKey(uiEvents, keyPresses)

	input, _, err := processInput("test processInput long", 0, 50, 1, AlwaysValid, uiEvents)
	if err != nil {
		t.Errorf("Error: %v", err)
	} else if input != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", input, testText)
	}
}

func TestDisplayResult(t *testing.T) {
	var longMsg []string
	for i := 0; i < 50; i++ {
		newLine := "Line " + strconv.Itoa(i)
		longMsg = append(longMsg, newLine)
	}

	for _, tt := range []struct {
		name  string
		msg   []string
		want  string
		human func(chan ui.Event, <-chan string)
	}{
		{
			name: "short_message",
			msg:  []string{"short message"},
			want: "short message",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"q"})
			},
		},
		{
			// Display the long message and immediately exit
			name: "long_message_press_esc",
			msg:  longMsg,
			want: strings.Join(longMsg[:resultHeight], "\n") + "\n\n(More)",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<Escape>"})
			},
		},
		{
			// Display the long message, scroll to the bottom, then exit
			name: "long message_scroll_to_end",
			msg:  longMsg,
			want: strings.Join(longMsg[len(longMsg)-resultHeight:], "\n") + "\n\n(End of message)",
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<PageDown>", "<PageDown>", "<PageDown>", "<Escape>"})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			menus := make(chan string)

			go tt.human(uiEvents, menus)
			msg, err := DisplayResult(tt.msg, uiEvents, menus)

			if err != nil && err != BackRequest {
				t.Errorf("Error: %v", err)
			}
			if tt.want != msg {
				t.Errorf("Incorrect value for msg. got: %v, want: %v", msg, tt.want)
			}

		})
	}
}

func TestCountNewlines(t *testing.T) {
	for _, tt := range []struct {
		name string
		str  string
		want int
	}{
		{
			name: "empty_string",
			str:  "",
			want: 0,
		},
		{
			name: "no_newline",
			str:  "test string",
			want: 0,
		},
		{
			name: "single_newline_end",
			str:  "test line\n",
			want: 1,
		},
		{
			name: "double_newline_end",
			str:  "test line\n\n",
			want: 2,
		},
		{
			name: "two_lines",
			str:  "test line 1\n test line 2\n",
			want: 2,
		},
		{
			name: "prefix_double_newline",
			str:  "\n\n test line 2",
			want: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lines := countNewlines(tt.str)
			if lines != tt.want {
				t.Errorf("Expected %d counted lines, but got %d\n", tt.want, lines)
			}
		})
	}
}

func TestPromptConfirmation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		wantBool bool
		wantErr  error
		human    func(chan ui.Event, <-chan string)
	}{
		{
			name:     "select_yes",
			wantBool: true,
			wantErr:  nil,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name:     "select_no",
			wantBool: true,
			wantErr:  nil,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name:     "go_back",
			wantBool: false,
			wantErr:  BackRequest,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<Escape>", "<Enter>"})
			},
		},
		{
			name:     "exit",
			wantBool: false,
			wantErr:  ExitRequest,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<C-d>", "<Enter>"})
			},
		},
		{
			name:     "change_response",
			wantBool: true,
			wantErr:  nil,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"1", "<Backspace>", "0", "<Enter>"})
			},
		},
		{
			name:     "submit_without_value",
			wantBool: true,
			wantErr:  nil,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"1", "<Backspace>", "0", "<Enter>"})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			menus := make(chan string)

			go tt.human(uiEvents, menus)
			accept, err := PromptConfirmation("Continue?", uiEvents, menus)
			if accept != tt.wantBool {
				t.Errorf("Expected %t, but received %t.\n", tt.wantBool, accept)
			} else if err != nil && err != tt.wantErr {
				t.Errorf("Expected error %v, but got %v", tt.wantErr, err)
			}
		})
	}
}

func TestDisplayMenu(t *testing.T) {
	entry1 := &testEntry{label: "entry 1"}
	entry2 := &testEntry{label: "entry 2"}
	entry3 := &testEntry{label: "entry 3"}
	entry4 := &testEntry{label: "entry 4"}
	entry5 := &testEntry{label: "entry 5"}
	entry6 := &testEntry{label: "entry 6"}
	entry7 := &testEntry{label: "entry 7"}
	entry8 := &testEntry{label: "entry 8"}
	entry9 := &testEntry{label: "entry 9"}
	entry10 := &testEntry{label: "entry 10"}
	entry11 := &testEntry{label: "entry 11"}
	entry12 := &testEntry{label: "entry 12"}

	for _, tt := range []struct {
		name    string
		entries []Entry
		want    Entry
		human   func(chan ui.Event, <-chan string)
	}{
		{
			name:    "hit_0",
			entries: []Entry{entry1, entry2, entry3},
			want:    entry1,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "<Enter>"})
			},
		},
		{
			name:    "hit_1",
			entries: []Entry{entry1, entry2, entry3},
			want:    entry2,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"1", "<Enter>"})
			},
		},
		{
			name:    "hit_2",
			entries: []Entry{entry1, entry2, entry3},
			want:    entry3,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"2", "<Enter>"})
			},
		},
		{
			name:    "error_input_then_right_input",
			entries: []Entry{entry1, entry2, entry3},
			want:    entry2,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"0", "a", "<Enter>", "1", "<Enter>"})
			},
		},
		{
			name:    "exceed_the_bound_then_right_input",
			entries: []Entry{entry1, entry2, entry3},
			want:    entry1,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"4", "<Enter>", "0", "<Enter>"})
			},
		},
		{
			name:    "right_input_with_backspace",
			entries: []Entry{entry1, entry2, entry3},
			want:    entry3,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"2", "a", "<Backspace>", "<Enter>"})
			},
		},
		{
			name:    "<pageDown>_<pageUp>_<pageDown>_hit_11",
			entries: []Entry{entry1, entry2, entry3, entry4, entry5, entry6, entry7, entry8, entry9, entry10, entry11, entry12},
			// hit <pageDown> -> <pageUp> -> <pageDown> current page is : 0~9
			want: entry12,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<PageDown>", "<pageUp>", "<PageDown>", "1", "1", "<Enter>"})
			},
		},
		{
			name:    "<Left>_<Right>_exceed_the_bound_then_right_input",
			entries: []Entry{entry1, entry2, entry3, entry4, entry5, entry6, entry7, entry8, entry9, entry10, entry11, entry12},
			// hit <Left> -> <Right> current page is : 10~11 because the first <Left> should do nothing
			want: entry11,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<Left>", "<Right>", "-", "1", "<Enter>", "1", "0", "<Enter>"})
			},
		},
		{
			name:    "<Down>_<Down>_<Up>_exceed_the_bound_then_right_input",
			entries: []Entry{entry1, entry2, entry3, entry4, entry5, entry6, entry7, entry8, entry9, entry10, entry11, entry12},
			// hit <Down> -> <Down> -> <Up> current page is : 1~10
			want: entry2,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<Down>", "<Down>", "<Up>", "2", "1", "<Enter>", "1", "<Enter>"})
			},
		},
		{
			name:    "<Down>_<End>_then_right_input",
			entries: []Entry{entry1, entry2, entry3, entry4, entry5, entry6, entry7, entry8, entry9, entry10, entry11, entry12},
			// hit <Down> -> <End> current page is : 2~11 because the <End> will move to the last page
			want: entry5,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<Down", "<End>", "4", "<Enter>"})
			},
		},
		{
			name:    "<Down>_<Home>_then_right_input",
			entries: []Entry{entry1, entry2, entry3, entry4, entry5, entry6, entry7, entry8, entry9, entry10, entry11, entry12},
			// hit <Down> -> <Home> current page is : 0~9 because the <End> will move to the first page
			want: entry1,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<Down>", "<Home>", "0", "<Enter>"})
			},
		},
		{
			name:    "<MouseWheelDown>_<MouseWheelDown>_<MouseWheelUp>_then_right_input",
			entries: []Entry{entry1, entry2, entry3, entry4, entry5, entry6, entry7, entry8, entry9, entry10, entry11, entry12},
			// scroll mouse wheel <MouseWheelDown> -> <MouseWheelDown> -> <MouseWheelUp> current page is : 1~10
			want: entry11,
			human: func(uiEvents chan ui.Event, menus <-chan string) {
				nextMenuReady(menus)
				pressKey(uiEvents, []string{"<MouseWheelDown>", "<MouseWheelDown>", "<MouseWheelUp>", "10", "<Enter>"})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			uiEvents := make(chan ui.Event)
			menus := make(chan string)

			//1go pressKey(uiEvents, tt.userInput)
			go tt.human(uiEvents, menus)

			chosen, err := PromptMenuEntry("test menu title", tt.name, tt.entries, uiEvents, menus)

			if err != nil {
				t.Errorf("Error: %v", err)
			}
			if tt.want != chosen {
				t.Errorf("Incorrect choice. Choose %+v, want %+v", chosen, tt.want)
			}

		})
	}
}
