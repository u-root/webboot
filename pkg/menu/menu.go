package menu

import (
	"fmt"
	"io"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const menuWidth = 50

// newParagraph returns a widgets.Paragraph struct with given initial text..
func newParagraph(initText string, border bool, location int, wid int, ht int) *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = initText
	p.Border = border
	p.SetRect(0, location, wid, location+ht)
	p.TextStyle.Fg = ui.ColorWhite
	return p
}

// readKey reads a key from input stream
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
func processInput(introwords string, location int, wid int, ht int, isValid func(string) (string, bool), uiEvents <-chan ui.Event) (string, string, error) {
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
			warningWords, ok := isValid(input.Text)
			if ok {
				return input.Text, warning.Text, nil
			}
			input.Text = ""
			warning.Text = warningWords
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

// NewCustomInputWindow creates a new ui window and displays an input box.
func NewCustomInputWindow(introwords string, wid int, ht int, isValid func(string) (string, bool)) (string, error) {
	uiEvents := ui.PollEvents()
	return newInputWindow(introwords, wid, ht, isValid, uiEvents)
}

// NewInputWindow opens a new input window with fixed width=100, hight=1
func NewInputWindow(introwords string, isValid func(string) (string, bool)) (string, error) {
	uiEvents := ui.PollEvents()
	return newInputWindow(introwords, 100, 1, isValid, uiEvents)
}

func newInputWindow(introwords string, wid int, ht int, isValid func(string) (string, bool), uiEvents <-chan ui.Event) (string, error) {
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
	uiEvents := ui.PollEvents()
	return displayResult(message, wid, uiEvents)
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
