package menu

import (
	"strconv"
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
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

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

func TestProcessInputSimple(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

	testText := "test"
	uiEvents := make(chan ui.Event)
	go pressKey(uiEvents, []string{"t", "e", "s", "t", "<Enter>"})

	input, warning, err := processInput("test processInput simple", 0, 50, 1, AlwaysValid, uiEvents)

	if err != nil {
		t.Errorf("ProcessInput failed: %v", err)
	}
	if input != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", input, testText)
	}
	if warning != "" {
		t.Errorf("Incorrect value for warning. got: %v, want nothing", warning)
	}
}

func TestProcessInputComplex(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

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

	input, warning, err := processInput("test processInput complex", 0, 50, 1, isValid, uiEvents)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if input != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", input, testText)
	}
	if warning != "" {
		t.Errorf("Incorrect value for warning. got: %v, want nothing", warning)
	}
}

func TestDisplayResult(t *testing.T) {
	testText := "test"
	uiEvents := make(chan ui.Event)
	go pressKey(uiEvents, []string{"q"})

	message := []string{"This is", "a", "TEST"}
	testText = "This is\na\nTEST"
	msg, err := DisplayResult(message, uiEvents)

	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if testText != msg {
		t.Errorf("Incorrect value for msg. got: %v, want: %v", msg, testText)
	}
}

func TestDisplayMenu(t *testing.T) {
	entry1 := &testEntry{label: "entry 1"}
	entry2 := &testEntry{label: "entry 2"}
	entry3 := &testEntry{label: "entry 3"}

	for _, tt := range []struct {
		name      string
		entries   []Entry
		userInput []string
		want      Entry
	}{
		{
			name:      "hit_enter",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"<Enter>"},
			want:      entry1,
		},
		{
			name:      "hit_0",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"0", "<Enter>"},
			want:      entry1,
		},
		{
			name:      "hit_1",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"1", "<Enter>"},
			want:      entry2,
		},
		{
			name:      "hit_2",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"2", "<Enter>"},
			want:      entry3,
		},
		{
			name:      "error_input_then_hit_enter",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"0", "a", "<Enter>", "<Enter>"},
			want:      entry1,
		},
		{
			name:      "exceed_the_bound_then_right_input",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"4", "<Enter>", "0", "<Enter>"},
			want:      entry1,
		},
		{
			name:      "right_input_with_backspace",
			entries:   []Entry{entry1, entry2, entry3},
			userInput: []string{"2", "a", "<Backspace>", "<Enter>"},
			want:      entry3,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uiEvents := make(chan ui.Event)
			go pressKey(uiEvents, tt.userInput)

			chosen, err := DisplayMenu("test menu title", tt.name, tt.entries, uiEvents)

			if err != nil {
				t.Errorf("Error: %v", err)
			}
			if tt.want != chosen {
				t.Errorf("Incorrect choice. Choose %+v, want %+v", chosen, tt.want)
			}

		})
	}

}
