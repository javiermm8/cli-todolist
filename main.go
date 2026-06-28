package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"slices"

	ui "github.com/metaspartan/gotui/v5"
	"github.com/metaspartan/gotui/v5/widgets"
)

// Structs(mainly for JSON)
type TODO struct {
	Title string
}

type TODOList struct {
	Name  string
	TODOs []TODO
}

func main() {
	var weOnFirstPage = true
	var noTODOList bool

	// Variable with TODOList as its struct
	var main TODOList

	// Open(or create it if it doesn't exist) .cli-TODO.json
	f, err := os.OpenFile(".cli-TODO.json", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Read .cli-TODO.json (output in bytes)
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the gotui ui
	if err := ui.Init(); err != nil {
		log.Fatalf("Failed to initialize gotui: %v", err)
	}
	defer ui.Close()

	// Create new paragraphs and set some info
	p := widgets.NewParagraph()
	p.Title = "Welcome!"
	p.TitleStyle.Fg = ui.ColorYellow
	p.BorderStyle.Fg = ui.ColorSkyBlue

	// Create input
	i := widgets.NewInput()

	// If the file is empty main is empty too, else put the json into the variable main(of struct TODOList)
	if len(data) == 0 {
		noTODOList = true
		main = TODOList{
			TODOs: []TODO{},
		}

		// Set p info
		p.Text = "No todolist found. Let's create one!\n\n(Press q to quit)"
		p.SetRect(0, 0, 50, 5)

		// Set input's info
		i.Title = "TODO List name:"
		i.Placeholder = "Enter TODO List name"
		i.SetRect(0, 5, 50, 8)

		// Render everything
		ui.Render(p, i)

	} else {
		// Put json data into main
		if err := json.Unmarshal(data, &main); err != nil {
			log.Fatal(err)
		}

		// Set ps info
		p.Text = "Todolist found: " + main.Name + "\n\n(Press <Enter> to continue or q to quit)"
		p.SetRect(0, 0, 50, 5)

		// Render p
		ui.Render(p)
	}

	uiEvents := ui.PollEvents()
	for weOnFirstPage {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			// Close the file
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
			return
		case "<Enter>":
			if noTODOList {
				main.Name = i.Text
				draft, err := json.Marshal(main)
				if err != nil {
					log.Fatal(err)
				}
				f.Write(draft)

				noTODOList = false
				ui.Clear()
				weOnFirstPage = false
				todo(main, f, uiEvents)
			} else {
				ui.Clear()
				weOnFirstPage = false
				todo(main, f, uiEvents)
			}

		default:
			switch e.ID {
			case "<Backspace>":
				if noTODOList {
					i.Backspace()
				}
			case "<Left>":
				if noTODOList {
					i.MoveCursorLeft()
				}
			case "<Right>":
				if noTODOList {
					i.MoveCursorRight()
				}
			case "<Space>":
				if noTODOList {
					i.InsertRune(' ')
				}
			default:
				if noTODOList {
					if len(e.ID) == 1 {
						i.InsertRune([]rune(e.ID)[0])
					}
				}
			}
		}
		if noTODOList {
			ui.Render(p, i)
		}
	}
}

func todo(main TODOList, f *os.File, uiEvents <-chan ui.Event) {
	var currentlySelected int = -1
	var renderingList []*widgets.Paragraph

	// Load main's TODOs into renderingList and sets its dimensions/absolute positions
	for _, t := range main.TODOs {
		m := widgets.NewParagraph()
		m.BorderStyle.Fg = ui.ColorSkyBlue
		m.Text = t.Title
		renderingList = append(renderingList, m)
	}
	for i, entry := range renderingList {
		entry.SetRect(1, 4+i*3, 49, 7+i*3)
	}

	// create and set info for the big p and the input
	p := widgets.NewParagraph()
	p.Title = main.Name
	p.TitleStyle.Fg = ui.ColorYellow
	p.BorderStyle.Fg = ui.ColorSkyBlue
	p.SetRect(0, 0, 50, 5+len(renderingList)*3)
	newTODO := widgets.NewInput()
	newTODO.Title = "New TODO:"
	newTODO.TitleStyle.Fg = ui.ColorYellow
	newTODO.BorderStyle.Fg = ui.ColorSkyBlue
	newTODO.SetRect(1, 1, 49, 4)

	// Commands on the side
	pSide := widgets.NewParagraph()
	pSide.Title = "Press:"
	pSide.TitleStyle.Fg = ui.ColorYellow
	pSide.BorderStyle.Fg = ui.ColorSkyBlue
	pSide.Text = "<Enter> to add a new TODO\n<Up>/<Down> to move around your todolist\n<Backspace> to complete and delete a TODO\nq to quit"
	pSide.SetRect(51, 0, 75, 10)

	// Render everything
	ui.Render(p, newTODO, pSide)
	for i := 0; i < len(renderingList); i++ {
		ui.Render(renderingList[i])
	}

	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			// Saving flow
			main.TODOs = main.TODOs[:0] // empty the slice
			for i := 0; i < len(renderingList); i++ {
				var entry TODO
				entry.Title = renderingList[i].Text
				main.TODOs = append(main.TODOs, entry)
			}

			draft, err := json.Marshal(main)
			if err != nil {
				log.Fatal(err)
			}

			if err := f.Truncate(0); err != nil {
				log.Fatal(err)
			}
			if _, err = f.WriteAt(draft, 0); err != nil {
				log.Fatal(err)
			}

			// Close the file
			if err := f.Close(); err != nil {
				log.Fatal(err)
			}
			return
		// Honestly I'm not going to add comments to the selection logic. It was a mess to develop and most of this projects coding time was spent here. I'm traumatized. All of this because widget.NewList() was too ugly.
		case "<Down>":
			if currentlySelected < len(renderingList)-1 {
				if currentlySelected >= 0 {
					renderingList[currentlySelected].BorderStyle.Fg = ui.ColorSkyBlue
				}
				currentlySelected += 1
				renderingList[currentlySelected].BorderStyle.Fg = ui.ColorDarkRed
			}
		case "<Up>":
			if currentlySelected > 0 {
				renderingList[currentlySelected].BorderStyle.Fg = ui.ColorSkyBlue
				currentlySelected -= 1
				renderingList[currentlySelected].BorderStyle.Fg = ui.ColorDarkRed
			}
		case "<Enter>":
			if newTODO.Text == "" {
				break
			}
			m := widgets.NewParagraph()
			m.BorderStyle.Fg = ui.ColorSkyBlue
			m.Text = newTODO.Text
			newTODO.Text = ""
			renderingList = append([]*widgets.Paragraph{m}, renderingList...)
			p.SetRect(0, 0, 50, 5+len(renderingList)*3)
			if currentlySelected >= 0 {
				currentlySelected += 1
			}
			for i, entry := range renderingList {
				entry.SetRect(1, 4+i*3, 49, 7+i*3)
			}
		case "<Backspace>":
			if newTODO.Text == "" && currentlySelected >= 0 {
				deletedIndex := currentlySelected
				oldY := renderingList[deletedIndex].GetRect().Min.Y

				renderingList = slices.Delete(renderingList, deletedIndex, deletedIndex+1)
				p.SetRect(0, 0, 50, 5+len(renderingList)*3)

				if currentlySelected >= len(renderingList) {
					currentlySelected = len(renderingList) - 1
				}
				if currentlySelected >= 0 {
					renderingList[currentlySelected].BorderStyle.Fg = ui.ColorDarkRed
				}

				for i := deletedIndex; i < len(renderingList); i++ {
					renderingList[i].SetRect(1, oldY+(i-deletedIndex)*3, 49, oldY+3+(i-deletedIndex)*3)
				}
			} else {
				newTODO.Backspace()
			}
		case "<Left>":
			newTODO.MoveCursorLeft()
		case "<Right>":
			newTODO.MoveCursorRight()
		default:
			if len(e.ID) == 1 {
				newTODO.InsertRune([]rune(e.ID)[0])
			}
		}

		ui.Clear()
		ui.Render(p, newTODO, pSide)
		for i := 0; i < len(renderingList); i++ {
			ui.Render(renderingList[i])
		}
	}
}
