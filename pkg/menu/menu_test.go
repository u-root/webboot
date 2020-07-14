package menu

import (
	ui "github.com/gizak/termui/v3"
	"strconv"
	"testing"
)

func TestNewParagraph(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

	testText := "newParagraph test"
	p := newParagraph(testText, false, 0, 50, 3)
	if testText != p.Text {
		t.Errorf("incorrect value for p.Text. got: %v, want: %v", p.Text, testText)
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

func TestGetInputSimple(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

	input := newParagraph("", true, 0, 50, 3)
	warning := newParagraph("", false, 3, 50, 3)
	ui.Render(input)
	ui.Render(warning)
	testText := "test"

	uiEvents := make(chan ui.Event)
	go pressKey(uiEvents, []string{"t", "e", "s", "t", "<Enter>"})

	isValid := func(input string) (string, bool) {
		return "", true
	}

	inputText, err := processInput(input, warning, isValid, uiEvents)

	if err != nil {
		t.Errorf("ProcessInput failed: %v", err)
	}
	if inputText != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", inputText, testText)
	}
	if warning.Text != "" {
		t.Errorf("Incorrect value for warning. got: %v, want nothing", warning.Text)
	}
}

func TestGetInputComplicated(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

	input := newParagraph("", true, 0, 50, 3)
	warning := newParagraph("", false, 3, 50, 3)
	ui.Render(input)
	ui.Render(warning)
	testText := "100"

	uiEvents := make(chan ui.Event)
	// mock user input
	// first input is bad input "bad"
	// second input is bad input "100a"
	// third input is good input "100" but contain a process of typo then backspace
	// now the warning text should be ""
	go pressKey(uiEvents, []string{"b", "a", "d", "<Enter>",
		"1", "0", "0", "a", "<Enter>",
		"1", "0", "a", "<Backspace>", "0", "<Enter>"})

	isValid := func(input string) (string, bool) {
		if _, err := strconv.ParseUint(input, 10, 32); err != nil {
			return "Input is not a valid entry number.", false
		}
		return "", true
	}

	inputText, err := processInput(input, warning, isValid, uiEvents)

	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if inputText != testText {
		t.Errorf("Incorrect value for input. got: %v, want: %v", inputText, testText)
	}
	if warning.Text != "" {
		t.Errorf("Incorrect value for warning. got: %v, want nothing", warning.Text)
	}
}
