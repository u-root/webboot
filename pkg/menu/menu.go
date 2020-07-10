package menu

import (
	"os"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const menuWidth = 50

// newParagraph will return a paragraph object with given initial text.
func newParagraph(initText string, border bool, location int, wid int, ht int) *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Text = initText
	p.Border = border
	p.SetRect(0, location, wid, location+ht)
	p.TextStyle.Fg = ui.ColorWhite
	ui.Render(p)
	return p
}

// GetInput will present an input box to user and return the user's input.
// GetInout will check validation of input using isValid function.
func GetInput(introwords string, location int, wid int, ht int, isValid func(string) (bool, string)) (string, error) {
	// intro paragraph is to tell user what need to be input here
	newParagraph(introwords, false, location, len(introwords)+4, 3)
	location += 2

	// input paragraph is where user input
	input := newParagraph("", true, location, wid, ht+2)
	location += ht + 2

	// warning paragraph is to warn user their input is not valid
	warning := newParagraph("", false, location, wid, 3)
	ui.Render(warning)

	// keep tracking all input from user
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		if e.Type != ui.KeyboardEvent {
			continue
		}
        switch e.ID {
		// if user input q or control-c, exit the program.
		case "q", "<C-c>":
			ui.Close()
			os.Exit(1)
        // If user hit enter means he did his choice.
		// So check whether the input is valid or not by isValid function.
		// If isValid is nil, just return the text anyway.
		// If the input is not acceptable, show a warning.
		case "<Enter>":
            if isValid == nil {
				return input.Text, nil
			}
			valid, warningWords := isValid(input.Text)
			// if input is vilid, directly return the input itself.
			// else we should clear the input box and show warnings.
			if valid {
				return input.Text, nil
			}
			input.Text = ""
			warning.Text = warningWords
			ui.Render(input)
			ui.Render(warning)
		//If userhit backspace, remove the last input character
		case "<Backspace>":
			input.Text = input.Text[:len(input.Text)-1]
			ui.Render(input)
		// as long as user do not hit enter or q or control-c, assuming that user is still inputing his choice
		default:
			// clear warning when user input
			if warning.Text != "" {
				warning.Text = ""
				ui.Render(warning)
			}
			input.Text += e.ID
			ui.Render(input)
		}
	}
}
