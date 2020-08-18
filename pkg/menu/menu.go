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
const menuHeight = 12

type validCheck func(string) (string, string, bool)

// Entry contains all the information needed for a boot entry.
type Entry interface {
	// Label returns the string will show in menu.
	Label() string
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
		if e.Type == ui.KeyboardEvent || e.Type == ui.MouseEvent {
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
	warning := newParagraph("", false, location, wid, 15)

	ui.Render(intro)
	ui.Render(input)
	ui.Render(warning)

	// keep tracking all input from user
	for {
		k := readKey(uiEvents)
		warning.Text = ""
		ui.Render(warning)
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
			if len(input.Text) > 0 {
				input.Text = input.Text[:len(input.Text)-1]
				ui.Render(input)
			}
		case "<Escape>":
			return "<Esc>", "", nil
		case "<Space>":
			input.Text += " "
			ui.Render(input)
		default:
			// the termui use a string begin at '<' to represent some special keys
			// for example the 'F1' key will be parsed to "<F1>" string .
			// we should do nothing when meet these special keys, we only care about alphabets and digits.
			if k[0:1] != "<" {
				input.Text += k
				ui.Render(input)
			}
		}
	}
}

// NewInputWindow opens a new input window with fixed width=100, hight=1.
func NewInputWindow(introwords string, isValid validCheck, uiEvents <-chan ui.Event) (string, error) {
	return NewCustomInputWindow(introwords, 80, 1, isValid, uiEvents)
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
func DisplayResult(message []string, uiEvents <-chan ui.Event) (string, error) {
	if err := ui.Init(); err != nil {
		return "", fmt.Errorf("Failed to initialize termui: %v", err)
	}
	defer ui.Close()

	var wid int = menuWidth

	// if a message is longer then width of the window, split it to shorter lines
	text := []string{}
	for _, m := range message {
		for len(m) > wid {
			text = append(text, m[0:wid])
			m = m[wid:len(m)]
		}
		text = append(text, m)
	}

	p := widgets.NewParagraph()
	p.Text = strings.Join(text, "\n")
	p.Border = false
	p.SetRect(0, 0, wid+2, len(text)+3)
	p.TextStyle.Fg = ui.ColorWhite

	ui.Render(p)

	// press any key to exit this window
	k := readKey(uiEvents)
	if k == "<C-d>" {
		return p.Text, io.EOF
	}
	return p.Text, nil
}

// parsingMenuOption parses the user's operation in the menu page, such as page up, page down, selection. etc
func parsingMenuOption(labels []string, menu *widgets.List, input, warning *widgets.Paragraph, uiEvents <-chan ui.Event) (int, error) {

	if len(labels) == 0 {
		return 0, fmt.Errorf("No Entry in the menu")
	}

	menuTitle := menu.Title + "---%v/%v"

	// first, last always point to the first and last entry in current menu page
	first := 0
	last := min(10, len(labels))
	listData := labels[first:last]
	menu.Rows = listData
	menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
	ui.Render(menu)

	// keep tracking all input from user
	for {
		k := readKey(uiEvents)
		warning.Text = ""
		ui.Render(warning)
		switch k {
		case "<C-d>":
			return 0, io.EOF
		case "<Enter>":
			choose := input.Text
			input.Text = ""
			ui.Render(input)
			c, err := strconv.Atoi(choose)
			// input is vilid when:
			// 1.input is a number;
			// 2.the number does not exceed the index in the current page.
			if err == nil && c >= first && c < last {
				return c, nil
			}
			warning.Text = "Please enter a valid entry number."
			ui.Render(warning)
		case "<Backspace>":
			if len(input.Text) > 0 {
				input.Text = input.Text[:len(input.Text)-1]
				ui.Render(input)
			}
		case "<Left>", "<PageUp>":
			// page up
			first = max(0, first-10)
			last = min(first+10, len(labels))
			listData := labels[first:last]
			menu.Rows = listData
			menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
			ui.Render(menu)
		case "<Right>", "<PageDown>":
			// page down
			if first+10 >= len(labels) {
				continue
			}
			first = first + 10
			last = min(first+10, len(labels))
			listData := labels[first:last]
			menu.Rows = listData
			menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
			ui.Render(menu)
		case "<Up>", "<MouseWheelUp>":
			// move one line up
			first = max(0, first-1)
			last = min(first+10, len(labels))
			listData := labels[first:last]
			menu.Rows = listData
			menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
			ui.Render(menu)
		case "<Down>", "<MouseWheelDown>":
			// move one line down
			last = min(last+1, len(labels))
			first = max(0, last-10)
			listData := labels[first:last]
			menu.Rows = listData
			menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
			ui.Render(menu)
		case "<Home>":
			// first page
			first = 0
			last = min(first+10, len(labels))
			listData := labels[first:last]
			menu.Rows = listData
			menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
			ui.Render(menu)
		case "<End>":
			// last page
			last = len(labels)
			first = max(0, last-10)
			listData := labels[first:last]
			menu.Rows = listData
			menu.Title = fmt.Sprintf(menuTitle, first, len(labels))
			ui.Render(menu)
		case "<Space>":
			input.Text += " "
			ui.Render(input)
		default:
			// the termui use a string begin at '<' to represent some special keys
			// for example the 'F1' key will be parsed to "<F1>" string .
			// we should do nothing when meet these special keys, we only care about alphabets and digits.
			if k[0:1] != "<" {
				input.Text += k
				ui.Render(input)
			}
		}
	}
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
	menu := widgets.NewList()
	menu.Title = menuTitle
	// menus's hight always be 12, which could diplay 10 entrys in one page
	menu.SetRect(0, location, menuWidth, location+menuHeight)
	location += menuHeight
	menu.TextStyle.Fg = ui.ColorWhite

	intro := newParagraph(introwords, false, location, len(introwords)+4, 3)
	location += 2
	input := newParagraph("", true, location, menuWidth, 3)
	location += 3
	warning := newParagraph("", false, location, menuWidth, 3)

	ui.Render(intro)
	ui.Render(input)
	ui.Render(warning)

	chooseIndex, err := parsingMenuOption(listData, menu, input, warning, uiEvents)
	if err != nil {
		return nil, fmt.Errorf("Fail to get the choose from menu: %+v", err)
	}
	return entries[chooseIndex], nil
}
