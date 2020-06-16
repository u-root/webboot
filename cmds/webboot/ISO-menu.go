package main

import (
    "flag"
	"log"
    "strconv"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)


//the entry interface record all information of a menu choise
type Entry interface {
	Label() string // the label will show in menu
	Do() error  //do func will be run when the entry is chosen
	IsDefault() bool  // IsDefault is true means this entry will be hit by default, if there is many default choise, the first on in the list will be choose. the rest will be ignored.
}

//GetChoose will present all entries into a menu with numbers, so that user can choose from them.
func GetChoose(coord *int, introwords string, entries ...Entry) error{
    listData := []string{}//listData contains all choice's labels
    
    for i, e := range entries {
        listData = append(listData, fmt.Sprintf("[%d] %s", i+1, e.Label()))
	}

    l := widgets.NewList()//l is a List item which will present like a menu by termui package
	l.Title = "Menu"
	l.Rows = listData
	l.SetRect(0, *coord, 50, *coord+2+len(entries)) // design the size and location of the menu
    *coord += 4+len(entries)
	l.TextStyle.Fg = ui.ColorWhite
    ui.Render(l)      
    fmt.Printf("\n"+introwords+"\n\n")

    p1 := widgets.NewParagraph() //p1 is a paragraph for user to input their choice, but it is also used for showing warnings
	p1.Text = ""
	p1.Border = false
	p1.SetRect(0, *coord, 50, *coord+4) 
    *coord += 4
    p1.TextStyle.Fg = ui.ColorWhite
    ui.Render(p1)      
    
    tempFlag := false
	uiEvents := ui.PollEvents() // keep tracking all input from user
    for {
        e := <-uiEvents
        if(e.Type != ui.KeyboardEvent){ 
            continue
        }
        switch e.ID {
            case "q","<C-c>": // if user input q or control-c, exit the program. 
                return nil
            case "<Enter>": // if user hit enter means he did his choise, so check waht the input is. if the input is not acceptable, show a warning.
                if(tempFlag == false){ // this tempFlag is for recognizing if I need to clear this paragraph, so that user can input a new choice. And it is also for clearing warnings
                    p1.Text = ""
                    tempFlag = true
                }
                if(p1.Text=="") { // user input nothing means default choice is chosen
                    for _, en := range entries {
                        if en.IsDefault() {
                            if err := en.Do(); err != nil {
                                fmt.Printf("Failed to do %s: %v", en.Label(), err)
                                return err
                            }
		                }
	                }
                    return nil 
                }
                c, err := strconv.Atoi(p1.Text) // try convert the input into a vilid number
                if(err!=nil || c<0 || c>len(entries)){
                    p1.Text = fmt.Sprintf("%s is not a valid entry number.", p1.Text)
                    tempFlag = false
                    ui.Render(p1)
                } else {
                    if err := entries[c-1].Do(); err != nil {
                        fmt.Printf("Failed to do %s: %v", entries[c-1].Label(), err)
                        return err
                    }
                    return nil
                }
            default:// as long as user do not hit enter or q or control-c, assuming that user is still inputing his choise
                if(tempFlag == false){
                    p1.Text = ""
                    tempFlag = true
                }
                p1.Text += e.ID
                ui.Render(p1)        
        }
    }
    return nil
}

//GetInput will present an input window to user and return the user's input in to the input channel.
func GetInput(coord *int, input chan string) error { 
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
                return nil
            case "<Enter>":
                input <- p1.Text
                return nil
            default:
                p1.Text += e.ID
                ui.Render(p1)        
        }
    }
}

//save for later when we need an input operation
// func (InputSomething) Do(coord *int) error{
//     fmt.Printf("\n----------------------------\nyou choose the \"input\" option \n")
//     input := make(chan string);
//     go GetInput(coord, input)
//     inputString := <-input
//     fmt.Printf("\n\nyou input: %v \n", inputString)
//     return nil
// }


//the ISO type contains information of a iso file, incluing its name, its filepath, the path to its kernel and if it should be choose by default.
type ISO struct{
    name string
    path string
    kernel string
    isDefault bool
}

func (u ISO) IsDefault() bool {
    return u.isDefault
}

//Do func is what to do when an iso is chosen.
func (u ISO) Do() error{// these command below is only work for OSX now. I plan to add a linux version and decided which to use by GOOS. hope it will work.  
    ui.Close() // close the old window and open a new one
    ui.Init() 
    fmt.Println("mounting iso...") 
    cmd := exec.Command("hdiutil","attach","-nomount",u.path) 
    output, err := cmd.CombinedOutput();
    if  err != nil {
        // ui.Close()
        fmt.Println(err)
        // os.Exit(1)
    }
    diskName := strings.Fields(string(output))[0]
    diskFile := "/mnt/iso"
    fmt.Println("iso filename is "+u.path)
    fmt.Println("mounted disk name is "+diskName)
    cmd = exec.Command("mount","-t","cd9660",diskName,diskFile)
    output, err = cmd.CombinedOutput();
    if  err != nil {
        // ui.Close()
        fmt.Println(err)
        // os.Exit(1)
    }

    fmt.Println("iso mounted")
    fmt.Printf("mounted disk file is %s\n", diskFile)
    walkfunc := func(path string, info os.FileInfo, err error) error {
        if(path == filepath.Join(diskFile, u.kernel)) {
            fmt.Printf("Kernel at %s is found\n", u.kernel)
        }
        return nil
    }

    fmt.Println("\nfinding kernel...")
    filepath.Walk(diskFile, walkfunc)
    fmt.Printf("Kernel path is %s\n",u.kernel)

    fmt.Println("\nunmounting iso...")
    cmd = exec.Command("umount",diskName)
    output, err = cmd.CombinedOutput();
    if  err != nil {
        // ui.Close()
        fmt.Println(err)
        // os.Exit(1)
    }
    cmd = exec.Command("hdiutil","detach", diskName)
    output, err = cmd.CombinedOutput();
    if  err != nil {
        // ui.Close()
        fmt.Println(err)
        // os.Exit(1)
    }
    fmt.Printf("\nejecting disk...\n%s\n",string(output))
    return nil
}


func (u ISO) String() string {
    return fmt.Sprintf("%s, %s, %s, %v\n", u.name, u.path, u.kernel, u.isDefault)
}

func (u ISO) Label() string {
    return u.name
}

var (
    isoDir = flag.String("dir", ".", "set the iso directory path")
)


func main() {


    flag.Parse()

    //kernelPath is for hardcode the path to kernel. should be replaced by u-root later
    kernelPath := make(map[string]string)
    kernelPath["archlinux-2020.06.01-x86_64.iso"] = "arch/boot/x86_64/vmlinuz"
    kernelPath["TinyCore-11.1.iso"] = "boot/vmlinuz"

    var entries []Entry

    walkfunc := func(path string, info os.FileInfo, err error) error {
        if(info.IsDir() == false  && filepath.Ext(path) == ".iso") {
            kPath, ok := kernelPath[info.Name()]
            if (ok == true) {
                u := ISO{info.Name(), path, kPath, true}
                entries = append(entries, &u)
            } else {
                log.Println("unknown iso file %s", path)
            }
        }
        return nil
    }


    filepath.Walk(*isoDir, walkfunc)

    if err := ui.Init(); err != nil {
        log.Fatalf("failed to initialize termui: %v", err)
    }

    coord := 0
    GetChoose(&coord, "Choose an iso you want to boot (hit enter to choose the default - 1 is the default option) >", entries...)

    log.Printf("end")
    for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
            if(e.ID=="q" || e.ID=="<C-c>"){
                break
            }
		}
	}
    ui.Close()
	return
}