package main

import (
	"log"
    "strconv"
    "fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Entry interface {
	Label() string
	Do(*int) error
	IsDefault() bool
}


func GetChoose(coord *int, option chan Entry,  entries ...Entry) {
    listData := []string{}
    
    for i, e := range entries {
        listData = append(listData, fmt.Sprintf("[%d] %s", i+1, e.Label()))
	}
    l := widgets.NewList()
	l.Title = "Menu"
	l.Rows = listData
	l.SetRect(0, *coord, 50, *coord+5)
    *coord += 5
	l.TextStyle.Fg = ui.ColorWhite
    ui.Render(l)      
    fmt.Printf("\nChoose a menu option (hit enter to choose the default - 1 is the default option) >\n")

    p1 := widgets.NewParagraph()
	p1.Text = ""
	p1.Border = false
	p1.SetRect(0, *coord, 50, *coord+4)
    *coord += 4
    p1.TextStyle.Fg = ui.ColorWhite
    ui.Render(p1)      
    
    flag := false
	uiEvents := ui.PollEvents()
    for {
        e := <-uiEvents
        if(e.Type != ui.KeyboardEvent){
            continue
        }
        switch e.ID {
            case "q","<C-c>":
                option <- nil
                return
            case "<Enter>":
                if(flag == false){
                    p1.Text = ""
                    flag = true
                }
                if(p1.Text=="") {
                    for _, en := range entries {
		              if en.IsDefault() {
			             option <- en
                          return
		              }
	                }
                    option <- nil
                    return 
                }
                c, err := strconv.Atoi(p1.Text)
                if(err!=nil || c<0 || c>2){
                    p1.Text = fmt.Sprintf("%s is not a valid entry number.", p1.Text)
                    flag = false
                    ui.Render(p1)
                } else {
                    option <- entries[c-1]
                    return 
                }
            default:
                if(flag == false){
                    p1.Text = ""
                    flag = true
                }
                p1.Text += e.ID
                ui.Render(p1)        
        }
    }
}

func GetInput(coord *int, input chan string) {
    p1 := widgets.NewParagraph()
    p1.Title = "Input here"
	p1.Text = ""
	p1.SetRect(0, *coord, 50, *coord+3)
    *coord += 3
    p1.TextStyle.Fg = ui.ColorWhite
    ui.Render(p1)      
    

	uiEvents := ui.PollEvents()
    for {
        e := <-uiEvents
        if(e.Type != ui.KeyboardEvent){
            continue
        }
        switch e.ID {
            case "q","<C-c>":
                input <- ""
                return
            case "<Enter>":
                input <- p1.Text
                return 
            default:
                p1.Text += e.ID
                ui.Render(p1)        
        }
    }
}

type ChooseAgain struct{}
type InputSomething struct{}

func (ChooseAgain) IsDefault() bool {
    return true
}

func (ChooseAgain) Do(coord *int) error{
    var entries []Entry;
    entries = append(entries, &ChooseAgain{}, &InputSomething{})
	fmt.Printf("\n----------------------------\nyou choose the \"choose again\" option \n")
    option := make(chan Entry);
    go GetChoose(coord, option, entries...)
    entry := <-option
    *coord += 1
    if(entry == nil){
        return nil
    }
	if err := entry.Do(coord); err != nil {
		fmt.Printf("Failed to do %s: %v", entry.Label(), err)
	}
    return nil
}


func (ChooseAgain) String() string {
    return fmt.Sprintf("choose again option")
}

func (ChooseAgain) Label() string {
	return "choose again"
}


func (InputSomething) IsDefault() bool {
    return false
}

func (InputSomething) Do(coord *int) error{
    fmt.Printf("\n----------------------------\nyou choose the \"input\" option \n")
    input := make(chan string);
    go GetInput(coord, input)
    inputString := <-input
    fmt.Printf("\n\nyou input: %v \n", inputString)
    return nil
}


func (InputSomething) String() string {
    return fmt.Sprintf("input option")
}

func (InputSomething) Label() string {
	return "input"
}



func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	
    
    var entries []Entry;
    entries = append(entries, &ChooseAgain{}, &InputSomething{})
    
    coord := 0
    
    option := make(chan Entry);
    go GetChoose(&coord, option, entries...)
    entry := <-option
    if err := entry.Do(&coord); err != nil {
		fmt.Printf("Failed to do %s: %v", entry.Label(), err)
	}
    
    for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}
	return
}