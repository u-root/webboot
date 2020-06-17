package menus

import (
	"log"
    "strconv"
    "fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

//Entry contains all the information needed for a boot entry
type Entry interface {
    //label is the string will show in menu
	Label() string 
    //IsDefault is true means this entry will be hit by default, if there is many default choise, the first on in the list will be choose. the rest will be ignored. 
    IsDefault() bool  
}


//GetChoose will present all entries into a menu with numbers, so that user can choose from them.
func GetChoose(coord *int, introwords string, entries []Entry) (int, error){
    listData := []string{}//listData contains all choice's labels
    
    for i, e := range entries {
        listData = append(listData, fmt.Sprintf("[%d] %s", i+1, e.Label()))
	}

    //l is a List item which will present like a menu by termui package
    l := widgets.NewList()	
    l.Title = "Menu"
	l.Rows = listData
    // design the size and location of the menu
	l.SetRect(0, *coord, 50, *coord+2+len(entries)) 
    *coord += 4+len(entries)
	l.TextStyle.Fg = ui.ColorWhite
    ui.Render(l)      
    log.Printf("\n"+introwords+"\n\n")

    //p1 is a paragraph for user to input their choice, but it is also used for showing warnings
    p1 := widgets.NewParagraph() 
	p1.Text = ""
	p1.Border = false
	p1.SetRect(0, *coord, 50, *coord+4) 
    *coord += 4
    p1.TextStyle.Fg = ui.ColorWhite
    ui.Render(p1)      
    
    tempFlag := false
    // keep tracking all input from user
	uiEvents := ui.PollEvents() 
    for {
        e := <-uiEvents
        if(e.Type != ui.KeyboardEvent){ 
            continue
        }
        switch e.ID {
            // if user input q or control-c, exit the program. 
            case "q","<C-c>": 
                return -1, nil
            // if user hit enter means he did his choise, so check waht the input is. if the input is not acceptable, show a warning.
            case "<Enter>": 
                // this tempFlag is for recognizing if I need to clear this paragraph, so that user can input a new choice. And it is also for clearing warnings
                if(tempFlag == false){ 
                    p1.Text = ""
                    tempFlag = true
                }
                // user input nothing means default choice is chosen
                if(p1.Text=="") { 
                    for i, en := range entries {
                        if en.IsDefault() {
                            return i, nil
		                }
	                }
	                //if no one is default, choose the first one. 
                    return 0, nil 
                }
                // try convert the input into a vilid number
                c, err := strconv.Atoi(p1.Text) 
                if(err!=nil || c<1 || c>len(entries)){
                    p1.Text = fmt.Sprintf("%s is not a valid entry number.", p1.Text)
                    tempFlag = false
                    ui.Render(p1)
                } else {
                    return c-1,err
                }
            // as long as user do not hit enter or q or control-c, assuming that user is still inputing his choise
            default:
                if(tempFlag == false){
                    p1.Text = ""
                    tempFlag = true
                }
                p1.Text += e.ID
                ui.Render(p1)        
        }
    }
    return -1,nil
}

//GetInput will present an input window to user and return the user's input in to the input channel.
func GetInput(coord *int) (string, error) { 
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
                return "",nil
            case "<Enter>":
                return p1.Text,nil
            default:
                p1.Text += e.ID
                ui.Render(p1)        
        }
    }
}


//save for later when we need an input operation
// func (InputSomething) Do(coord *int) error{
//     fmt.Printf("\n----------------------------\nyou choose the \"input\" option \n")
//     inputString,err := GetInput(coord, input)
//     fmt.Printf("\n\nyou input: %v \n", inputString)
//     return nil
// }
