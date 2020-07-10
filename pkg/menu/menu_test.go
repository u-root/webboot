package menu

import (
	ui "github.com/gizak/termui/v3"
	"strconv"
	"testing"
)

func Test_newParagraph(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Log(err)
	}
	defer ui.Close()
	p := newParagraph("newParagraph test", false, 0, 50, 3)
	t.Log(p.Text)
}

func Test_GetInput(t *testing.T) {
	if err := ui.Init(); err != nil {
		t.Log(err)
	}
	defer ui.Close()
	isValid := func(input string) (bool, string) {
        c, err := strconv.Atoi(input)
		if err != nil || c < 0 {
			return false, "your input is not a valid entry number."
		}
		return true, ""
	}
	input, err := GetInput("GetInput test", 0, 50, 1, isValid)
	t.Log(input)
	if err != nil {
		t.Errorf("%v", err)
	}
}
