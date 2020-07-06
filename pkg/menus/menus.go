package menus

import (
	"fmt"
	"os"
	"strconv"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var menuWidth = 50

// Entry contains all the information needed for a boot entry
type Entry interface {
	// Label is the string will show in menu
	Label() string
	// IsDefault is true means this entry will be hit by default.
	// If there is many default choise, the first on in the list will be choose.
	// the rest will be ignored.
	IsDefault() bool
	Do() error
}

// DisplayMenu will present all entries into a menu with numbers, so that user can choose from them.
func DisplayMenu(menuTitle string, introwords string, location int, entries []Entry) (Entry, error) {
	// open a new window
	if err := ui.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize termui: %v", err)
	}
	// listData contains all choice's labels
	listData := []string{}

	for i, e := range entries {
		listData = append(listData, fmt.Sprintf("[%d] %s", i+1, e.Label()))
	}

	// l is a List item which will present like a menu by termui package
	l := widgets.NewList()
	l.Title = menuTitle
	l.Rows = listData
	// design the size and location of the menu
	l.SetRect(0, location, menuWidth, location+len(entries)+2)
	location += len(entries) + 2
	l.TextStyle.Fg = ui.ColorWhite
	ui.Render(l)

	// checkValidFunc is a function for GetInput to check the format of input.
	// Since we want user to choose a option from the menu, the checkValidFunction will check :
	// 1.input is a number; 2.input number does not exceed the number of options.
	checkValidFunc := func(input string) (bool, string) {
		if input == "" {
			for _, en := range entries {
				if en.IsDefault() {
					return true, ""
				}
			}
			return false, "no default option, please enter a choice"
		}
		c, err := strconv.Atoi(input)
		if err != nil || c < 1 || c > len(entries) {
			return false, "your input is not a valid entry number."
		}
		return true, ""
	}

	// call GetInput to get user's choice
	choose, err := GetInput(introwords, location, 100, 1, checkValidFunc)

	if err != nil {
		ui.Close()
		return nil, fmt.Errorf("Error at input of desplay menu: %v", err)
	}

	if choose == "" {
		for i, en := range entries {
			if en.IsDefault() {
				ui.Close()
				err = en.Do()
				return en, err
			}
		}
	} else {
		c, err := strconv.Atoi(choose)
		ui.Close()
		if err != nil {
			return nil, fmt.Errorf("Error at convert input to number in desplay menu: %v", err)
		}
		err = entries[c-1].Do()
		if err != nil {
			return nil, fmt.Errorf("Error at Do() in desplay menu: %v", err)
		}
		return entries[c-1], nil
	}
	ui.Close()
	return nil, nil
}

// GetInput will present an input box to user and return the user's input.
// GetInout will check validation of input using checkValidFunc.
func GetInput(introwords string, location int, wid int, ht int, checkValidFunc func(string) (bool, string)) (string, error) {
	// intro paragraph is to tell user what need to be input here
	intro := widgets.NewParagraph()
	intro.Text = introwords
	intro.Border = false
	intro.SetRect(0, location, len(introwords), location+3)
	intro.TextStyle.Fg = ui.ColorWhite
	ui.Render(intro)
	location += 3

	// input paragraph is where user input
	input := widgets.NewParagraph()
	input.Text = ""
	intro.Border = false
	input.SetRect(0, location, wid, location+ht+2)
	input.TextStyle.Fg = ui.ColorWhite
	ui.Render(input)
	location += ht + 2

	// warning paragraph is to warn user their input is not valid
	warning := widgets.NewParagraph()
	warning.Text = ""
	warning.Border = false
	warning.SetRect(0, location, wid, location+3)
	warning.TextStyle.Fg = ui.ColorWhite
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
			ui.Close()
			os.Exit(1)
		// if user hit enter means he did his choice.
		// So check whether the input is valid or not by checkVilidFunc function.
		// If the input is not acceptable, show a warning.
		case "<Enter>":
			valid, warningWords := checkValidFunc(input.Text)
			// if input is vilid, directly return the input itself.
			// else we should clear the input box and show warnings.
			if valid {
				return input.Text, nil
			}
			input.Text = ""
			warning.Text = warningWords
			ui.Render(input)
			ui.Render(warning)
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

// NewInputWindow will create a new ui window and display an input box.
// We have this function because GetInput function will display in the current window.
// And sometimes we want anew window to make the UI looks clean and neat.
func NewInputWindow(introwords string, wid int, ht int, checkValidFunc func(string) (bool, string)) (string, error) {
	if err := ui.Init(); err != nil {
		return "", fmt.Errorf("failed to initialize termui: %v", err)
	}
	input, err := GetInput(introwords, 0, wid, ht, checkValidFunc)
	ui.Close()
	return input, err
}
