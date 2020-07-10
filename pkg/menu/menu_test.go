package menu

import (
    "testing"
    ui "github.com/gizak/termui/v3"
)

func Test_newParagraph(t *testing.T){
    if err := ui.Init(); err != nil {
        t.Log(err)
    }
    defer ui.Close()
    p := newParagraph("test", false, 0, 50, 3)
    t.Log(p.Text)
}