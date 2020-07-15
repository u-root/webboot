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

func TestProcessInputSimple(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Fatal(err)
	}
	defer ui.Close()

	testText := "test"
	uiEvents := make(chan ui.Event)
	go pressKey(uiEvents, []string{"t", "e", "s", "t", "<Enter>"})

	isValid := func(input string) (string, bool) {
		return "", true
	}

	input, warning, err := processInput("test processInput simple", 0, 50, 1, isValid, uiEvents)

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

func TestNewCustomInputWindow(t *testing.T) {
	testText := "test"
	uiEvents := make(chan ui.Event)
	go pressKey(uiEvents, []string{"t", "e", "s", "t", "<Enter>"})

	isValid := func(input string) (string, bool) {
		return "", true
	}

	input, err := internalNewInputWindow("Test NewCustomInputWindow", 100, 2, isValid, uiEvents)

	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if testText != input {
		t.Errorf("incorrect value for input. got: %v, want: %v", input, testText)
	}
}
