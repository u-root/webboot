package menu

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const menuWidth = 50

type validCheck func(string) (string, string, bool)

// Entry contains all the information needed for a boot entry.
type Entry interface {
	// Label returns the string will show in menu.
	Label() string
}

// AlwaysValid is a special isValid function that check nothing
func AlwaysValid(input string) (string, string, bool) {
	return input, "", true
}

// newParagraph returns a widgets.Paragraph struct with given initial text.
func newParagraph(initText string, border bool, location int, wid int, ht int) *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = initText
	p.Border = border
	p.SetRect(0, location, wid, location+ht)
	p.TextStyle.Fg = ui.ColorWhite
	return p
}

// readKey reads a key from input stream.
func readKey(uiEvents <-chan ui.Event) string {
	for {
		e := <-uiEvents
		if e.Type == ui.KeyboardEvent {
			return e.ID
		}
	}
}

// processInput presents an input box to user and returns the user's input.
// processInput will check validation of input using isValid function.
func processInput(introwords string, location int, wid int, ht int, isValid validCheck, uiEvents <-chan ui.Event) (string, string, error) {
	intro := newParagraph(introwords, false, location, len(introwords)+4, 3)
	location += 2
	input := newParagraph("", true, location, wid, ht+2)
	location += ht + 2
	warning := newParagraph("", false, location, wid, 3)

	ui.Render(intro)
	ui.Render(input)
	ui.Render(warning)

	// keep tracking all input from user
	for {
		k := readKey(uiEvents)
		switch k {
		case "<C-d>":
			return input.Text, warning.Text, io.EOF
		case "<Enter>":
			inputString, warningString, ok := isValid(input.Text)
			if ok {
				return inputString, warning.Text, nil
			}
			input.Text = ""
			warning.Text = warningString
			ui.Render(input)
			ui.Render(warning)
		case "<Backspace>":
			if len(input.Text) == 0 {
				continue
			}
			input.Text = input.Text[:len(input.Text)-1]
			ui.Render(input)
		default:
			if warning.Text != "" {
				warning.Text = ""
				ui.Render(warning)
			}
			input.Text += k
			ui.Render(input)
		}
	}
}

// NewInputWindow opens a new input window with fixed width=100, hight=1.
func NewInputWindow(introwords string, isValid validCheck, uiEvents <-chan ui.Event) (string, error) {
	return NewCustomInputWindow(introwords, 100, 1, isValid, uiEvents)
}

// NewCustomInputWindow creates a new ui window and displays an input box.
func NewCustomInputWindow(introwords string, wid int, ht int, isValid validCheck, uiEvents <-chan ui.Event) (string, error) {
	if err := ui.Init(); err != nil {
		return "", fmt.Errorf("Failed to initialize termui: %v", err)
	}
	defer ui.Close()

	input, _, err := processInput(introwords, 0, wid, ht, isValid, uiEvents)

	return input, err
}

// DisplayResult opens a new window and displays a message.
// each item in the message array will be displayed on a single line.
func DisplayResult(message []string, wid int) (string, error) {
	return displayResult(message, wid, ui.PollEvents())
}

func displayResult(message []string, wid int, uiEvents <-chan ui.Event) (string, error) {
	if err := ui.Init(); err != nil {
		return "", fmt.Errorf("Failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Text = strings.Join(message, "\n")
	p.Border = false
	p.SetRect(0, 0, wid, len(message)+3)
	p.TextStyle.Fg = ui.ColorWhite

	ui.Render(p)

	// press any key to exit this window
	k := readKey(uiEvents)
	if k == "<C-d>" {
		return p.Text, io.EOF
	}
	return p.Text, nil
}

// DisplayMenu presents all entries into a menu with numbers.
// user inputs a number to choose from them.
func DisplayMenu(menuTitle string, introwords string, entries []Entry, uiEvents <-chan ui.Event) (Entry, error) {
	if err := ui.Init(); err != nil {
		return nil, fmt.Errorf("Failed to initialize termui: %v", err)
	}
	defer ui.Close()
	// listData contains all choice's labels
	listData := []string{}

	for i, e := range entries {
		listData = append(listData, fmt.Sprintf("[%d] %s", i, e.Label()))
	}

	location := 0
	l := widgets.NewList()
	l.Title = menuTitle
	l.Rows = listData
	l.SetRect(0, location, menuWidth, location+len(entries)+2)
	location += len(entries) + 2
	l.TextStyle.Fg = ui.ColorWhite
	ui.Render(l)

	// we want user to choose from the menu, so the isValid will check :
	// 1.input is a number; 2.input number does not exceed the number of options.
	isValid := func(input string) (string, string, bool) {
		if input == "" {
			if len(entries) > 0 {
				return "0", "", true
			}
			return "", "No default option, please enter a choice", false
		}
		if c, err := strconv.Atoi(input); err != nil || c < 0 || c >= len(entries) {
			return "", "Input is not a valid entry number.", false
		}
		return input, "", true
	}

	input, _, err := processInput(introwords, location, menuWidth, 1, isValid, uiEvents)

	if err != nil {
		return nil, fmt.Errorf("Failed to get input in displayMenu: %v", err)
	}

	choose, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert input to number in desplayMenu: %v", err)
	}
	return entries[choose], nil
}
