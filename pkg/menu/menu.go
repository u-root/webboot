package menu

import (
    ui "github.com/gizak/termui/v3"
    "github.com/gizak/termui/v3/widgets"
)

const menuWidth = 50

// newParagraph will return a paragraph object with given initial text.
func newParagraph(initText string, Border bool, location int, wid int, ht int) *widgets.Paragraph{
    p := widgets.NewParagraph()
    p.Text = initText
    p.Border = false
    p.SetRect(0, location, wid, location+ht)
    p.TextStyle.Fg = ui.ColorWhite
    ui.Render(p)
    return p
}

