package menu

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const menuWidth = 50
const menuHeight = 12
const resultHeight = 20
const resultWidth = 70

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

func countNewlines(str string) int {
	count := 0
	for _, s := range str {
		if s == '\n' {
			count++
		}
	}
	return count
}

func Init() error {
	return ui.Init()
}

func Close() {
	ui.Close()
}

var BackRequest = errors.New("User requested to return to a previous menu.")
var ExitRequest = errors.New("User requested to exit the program.")

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
	warning := newParagraph("<Esc> to go back, <Ctrl+d> to exit", false, location, wid, 15)

	ui.Render(intro)
	ui.Render(input)
	ui.Render(warning)

	// The input box is wid characters wide
	//   - 2 chars are reserved for the left and right borders
	//   - 1 char is left empty at the end of input to visually
	//        signify that the text box is still accepting input
	// The user might want to input a string longer than wid-3
	// characters, so we store the full typed input in fullText
	// and display a substring of the full text to the user
	var fullText string

	for {
		k := readKey(uiEvents)
		switch k {
		case "<C-d>":
			return "", "", ExitRequest
		case "<Escape>":
			return "", "", BackRequest
		case "<Enter>":
			inputString, warningString, ok := isValid(fullText)
			if ok {
				return inputString, warning.Text, nil
			}
			fullText = ""
			input.Text = ""
			warning.Text = warningString
			ui.Render(input)
			ui.Render(warning)
		case "<Backspace>":
			if len(input.Text) > 0 {
				fullText = fullText[:len(fullText)-1]
				start := max(0, len(fullText)-wid+3)
				input.Text = fullText[start:len(fullText)]
				ui.Render(input)
			}
		case "<Space>":
			fullText += " "
			start := max(0, len(fullText)-wid+3)
			input.Text = fullText[start:len(fullText)]
			ui.Render(input)
		default:
			// the termui use a string begin at '<' to represent some special keys
			// for example the 'F1' key will be parsed to "<F1>" string .
			// we should do nothing when meet these special keys, we only care about alphabets and digits.
			if k[0:1] != "<" {
				fullText += k
				start := max(0, len(fullText)-wid+3)
				input.Text = fullText[start:len(fullText)]
				ui.Render(input)
			}
		}
	}
}

// PromptTextInput opens a new input window with fixed width=100, hight=1.
func PromptTextInput(introwords string, isValid validCheck, uiEvents <-chan ui.Event) (string, error) {
	defer ui.Clear()
	input, _, err := processInput(introwords, 0, 80, 1, isValid, uiEvents)
	return input, err
}

// DisplayResult opens a new window and displays a message.
// each item in the message array will be displayed on a single line.
func DisplayResult(message []string, uiEvents <-chan ui.Event) (string, error) {
	defer ui.Clear()

	// if a message is longer then width of the window, split it to shorter lines
	var wid int = resultWidth
	text := []string{}
	for _, m := range message {
		for len(m) > wid {
			text = append(text, m[0:wid])
			m = m[wid:len(m)]
		}
		text = append(text, m)
	}

	p := widgets.NewParagraph()
	p.Border = true
	p.SetRect(0, 0, resultWidth+2, resultHeight+4)
	p.TextStyle.Fg = ui.ColorWhite

	msgLength := len(text)
	first := 0
	last := min(resultHeight, msgLength)

	controlText := "<Page Up>, <Page Down> to scroll\n\nPress any other key to continue."
	controls := newParagraph(controlText, false, resultHeight+4, wid+2, 5)
	ui.Render(controls)

	for {
		p.Title = fmt.Sprintf("Message---%v/%v", first, msgLength)
		displayText := strings.Join(text[first:last], "\n")

		// Indicate whether user is at the
		// end of text for long messages
		if msgLength > resultHeight {
			if last < msgLength {
				displayText += "\n\n(More)"
			} else if last == msgLength {
				displayText += "\n\n(End of message)"
			}
		}

		p.Text = displayText
		ui.Render(p)

		k := readKey(uiEvents)
		switch k {
		case "<Up>", "<MouseWheelUp>":
			first = max(0, first-1)
			last = min(first+resultHeight, len(text))
		case "<Down>", "<MouseWheelDown>":
			last = min(last+1, len(text))
			first = max(0, last-resultHeight)
		case "<Left>", "<PageUp>":
			first = max(0, first-resultHeight)
			last = min(first+resultHeight, len(text))
		case "<Right>", "<PageDown>":
			last = min(last+resultHeight, len(text))
			first = max(0, last-resultHeight)
		case "<C-d>":
			return p.Text, ExitRequest
		case "<Escape>":
			return p.Text, BackRequest
		default:
			return p.Text, nil
		}
	}
}

// parsingMenuOption parses the user's operation in the menu page, such as page up, page down, selection. etc
func parsingMenuOption(labels []string, menu *widgets.List, input, warning *widgets.Paragraph, uiEvents <-chan ui.Event, customWarning ...string) (int, error) {

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
		switch k {
		case "<C-d>":
			return -1, ExitRequest
		case "<Escape>":
			return -1, BackRequest
		case "<Enter>":
			choose := input.Text
			input.Text = ""
			ui.Render(input)
			c, err := strconv.Atoi(choose)
			// Input is valid if the selected index
			// is between 0 <= input < len(labels)
			if err == nil && c >= 0 && c < len(labels) {
				// if there is not specific warning for this entry, return it
				// elsewise show the warning and continue
				if len(customWarning) > c && customWarning[c] != "" {
					warning.Text = customWarning[c]
					ui.Render(warning)
					continue
				}
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

// PromptMenuEntry presents all entries into a menu with numbers.
// user inputs a number to choose from them.
// customWarning allow self-defined warnings in the menu
// for example the wifi menu want to show specific warning when user hit a specific entry,
// because some wifi's type may not be supported.
func PromptMenuEntry(menuTitle string, introwords string, entries []Entry, uiEvents <-chan ui.Event, customWarning ...string) (Entry, error) {
	defer ui.Clear()

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
	warning := newParagraph("<Esc> to go back, <Ctrl+d> to exit", false, location, menuWidth, 3)

	ui.Render(intro)
	ui.Render(input)
	ui.Render(warning)

	chooseIndex, err := parsingMenuOption(listData, menu, input, warning, uiEvents, customWarning...)
	if err != nil {
		return nil, err
	}

	return entries[chooseIndex], nil
}

func PromptConfirmation(message string, uiEvents <-chan ui.Event) (bool, error) {
	defer ui.Clear()

	wid := resultWidth
	text := ""
	position := 1

	for {
		// Split message if longer than msg box
		end := min(len(message), wid)
		text += message[:end] + "\n"
		if len(message) > wid {
			message = message[wid:]
		} else {
			break
		}
	}

	text += "\n[0] Yes\n[1] No\n"
	position += countNewlines(text) + 2
	wid += 2 // 2 borders

	msg := newParagraph(text, true, 0, wid, position)
	ui.Render(msg)

	selectHint := newParagraph("Choose an option:", false, position+1, wid, 1)
	ui.Render(selectHint)

	entry := newParagraph("", true, position+2, wid, 3)
	ui.Render(entry)

	backHint := newParagraph("<Esc> to go back, <Ctrl+d> to exit", false, position+6, wid, 1)
	ui.Render(backHint)

	for {
		key := readKey(uiEvents)
		switch key {
		case "<Escape>":
			return false, BackRequest
		case "<C-d>":
			return false, ExitRequest
		case "<Enter>":
			switch entry.Text {
			case "0":
				return true, nil
			case "1":
				return false, nil
			}
		case "0", "1":
			entry.Text = key
			ui.Render(entry)
		case "<Backspace>":
			entry.Text = ""
			ui.Render(entry)
		}
	}
}

type Progress struct {
	paragraph *widgets.Paragraph
	animated  bool
	sigTerm   chan bool
	ackTerm   chan bool
}

func NewProgress(text string, animated bool) Progress {
	paragraph := widgets.NewParagraph()
	paragraph.Border = true
	paragraph.SetRect(0, 0, resultWidth, 10)
	paragraph.TextStyle.Fg = ui.ColorWhite
	paragraph.Title = "Operation Running"
	paragraph.Text = text
	ui.Render(paragraph)

	progress := Progress{paragraph, animated, make(chan bool), make(chan bool)}
	if animated {
		go progress.animate()
	}
	return progress
}

func (p *Progress) Update(text string) {
	p.paragraph.Text = text
	ui.Render(p.paragraph)
}

func (p *Progress) animate() {
	counter := 0
	for {
		select {
		case <-p.sigTerm:
			p.ackTerm <- true
			return
		default:
			time.Sleep(time.Second)
			pText := p.paragraph.Text
			p.Update(pText + strings.Repeat(".", counter%4))
			p.paragraph.Text = pText
			counter++
		}
	}
}

func (p *Progress) Close() {
	if p.animated {
		p.sigTerm <- true
		<-p.ackTerm
	}
	ui.Clear()
}
