package main

import (
	"encoding/json"
	"fmt"
	"io"
	"image/color"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SudokuGrid struct {
	Grid [][]int `json:"grid"`
}

type GameState struct {
	Values [9][9]int     `json:"values"`
	Locked [9][9]bool    `json:"locked"`
	Notes  [9][9][9]bool `json:"notes"`
}

type Cell struct {
	widget.BaseWidget
	row, col int
	val      int
	notes    [9]bool
	isLocked bool
	onSelect func(r, c int)
	selected bool
	hovered  bool
}

func NewCell(r, c, val int, isLocked bool, onSelect func(r, c int)) *Cell {
	cell := &Cell{row: r, col: c, val: val, isLocked: isLocked, onSelect: onSelect}
	cell.ExtendBaseWidget(cell)
	return cell
}

func (c *Cell) CreateRenderer() fyne.WidgetRenderer {
	// Use a color that is almost transparent but still hit-testable
	bg := canvas.NewRectangle(color.NRGBA{0, 0, 0, 1})
	bg.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30}
	bg.StrokeWidth = 1

	mainText := canvas.NewText("", color.White)
	mainText.Alignment = fyne.TextAlignCenter
	mainText.TextStyle = fyne.TextStyle{Bold: true}

	noteTexts := make([]*canvas.Text, 9)
	noteContainer := container.New(layout.NewGridLayout(3))
	for i := 0; i < 9; i++ {
		t := canvas.NewText("", color.Gray{Y: 150})
		t.Alignment = fyne.TextAlignCenter
		noteTexts[i] = t
		noteContainer.Add(t)
	}

	return &cellRenderer{
		cell:          c,
		bg:            bg,
		mainText:      mainText,
		noteTexts:     noteTexts,
		noteContainer: noteContainer,
		objects:       []fyne.CanvasObject{bg, mainText, noteContainer},
	}
}

type cellRenderer struct {
	cell          *Cell
	bg            *canvas.Rectangle
	mainText      *canvas.Text
	noteTexts     []*canvas.Text
	noteContainer *fyne.Container
	objects       []fyne.CanvasObject
}

func (r *cellRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.mainText.Resize(size)
	r.mainText.TextSize = size.Height * 0.7
	r.noteContainer.Resize(size)
	noteSize := size.Height * 0.22
	for _, t := range r.noteTexts {
		t.TextSize = noteSize
	}
}

func (r *cellRenderer) MinSize() fyne.Size {
	return fyne.NewSize(30, 30) // Slightly larger min size
}

func (r *cellRenderer) Refresh() {
	if r.cell.selected {
		r.bg.FillColor = color.NRGBA{R: 80, G: 80, B: 200, A: 120} // Slightly more opaque for better visibility
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
	} else if r.cell.hovered {
		r.bg.FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30}
		r.bg.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 100}
		r.bg.StrokeWidth = 1
	} else {
		r.bg.FillColor = color.NRGBA{0, 0, 0, 1} // Almost transparent but solid for hit-testing
		r.bg.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30}
		r.bg.StrokeWidth = 1
	}

	if r.cell.val > 0 {
		r.mainText.Text = strconv.Itoa(r.cell.val)
		if r.cell.isLocked {
			r.mainText.Color = color.NRGBA{R: 255, G: 215, B: 0, A: 255} // Gold for clues
			r.mainText.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			r.mainText.Color = color.White
			r.mainText.TextStyle = fyne.TextStyle{}
		}
		r.mainText.Show()
		r.noteContainer.Hide()
	} else {
		r.mainText.Hide()
		r.noteContainer.Show()
		for i := 0; i < 9; i++ {
			if r.cell.notes[i] {
				r.noteTexts[i].Text = strconv.Itoa(i + 1)
			} else {
				r.noteTexts[i].Text = ""
			}
		}
	}
	r.bg.Refresh()
}

func (r *cellRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *cellRenderer) Destroy()                     {}

func (c *Cell) Tapped(_ *fyne.PointEvent) {
	c.onSelect(c.row, c.col)
}

func (c *Cell) MouseIn(_ *desktop.MouseEvent) {
	c.hovered = true
	c.Refresh()
}

func (c *Cell) MouseOut() {
	c.hovered = false
	c.Refresh()
}

func (c *Cell) MouseMoved(_ *desktop.MouseEvent) {}

func main() {
	fmt.Println("Sudoku Helper starting...")
	var initialGrid [][]int
	var baseName string
	var saveFileName string

	parseGrid := func(raw []byte) [][]int {
		str := string(raw)
		// Improved regex to find the JSON block even inside markdown
		re := regexp.MustCompile(`(?s)\{\s*"grid"\s*:\s*\[.*\]\s*\}`)
		match := re.FindString(str)
		if match == "" {
			return nil
		}
		var sg SudokuGrid
		if err := json.Unmarshal([]byte(match), &sg); err != nil {
			return nil
		}
		return sg.Grid
	}

	if len(os.Args) >= 2 {
		imgPath := os.Args[1]
		baseName = strings.TrimSuffix(filepath.Base(imgPath), filepath.Ext(imgPath))
		saveFileName = baseName + "_savegame.json"

		data, err := os.ReadFile(imgPath)
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
		initialGrid = parseGrid(data)
		if initialGrid == nil {
			log.Fatalf("Could not parse initial grid from %s", imgPath)
		}
	} else {
		baseName = "new_game"
		saveFileName = "new_game_savegame.json"
		initialGrid = make([][]int, 9)
		for i := range initialGrid {
			initialGrid[i] = make([]int, 9)
		}
	}

	myApp := app.NewWithID("com.gemini.sudoku")
	myApp.Settings().SetTheme(theme.DarkTheme())
	myWindow := myApp.NewWindow("Sudoku Visualizer")
	myWindow.SetIcon(resourceIconPng)

	cells := [9][9]*Cell{}
	var selectedR, selectedC int = -1, -1
	var bigGrid *fyne.Container

	onSelect := func(r, c int) {
		if selectedR != -1 {
			cells[selectedR][selectedC].selected = false
			cells[selectedR][selectedC].Refresh()
		}
		selectedR, selectedC = r, c
		cells[r][c].selected = true
		cells[r][c].Refresh()
	}

	isValidMove := func(grid [9][9]int, r, c, num int) bool {
		for col := 0; col < 9; col++ {
			if col != c && grid[r][col] == num {
				return false
			}
		}
		for row := 0; row < 9; row++ {
			if row != r && grid[row][c] == num {
				return false
			}
		}
		startR, startC := (r/3)*3, (c/3)*3
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if (startR+i != r || startC+j != c) && grid[startR+i][startC+j] == num {
					return false
				}
			}
		}
		return true
	}

	clearConflictingNotes := func(r, c, num int) {
		for col := 0; col < 9; col++ {
			if col != c && cells[r][col].notes[num-1] {
				cells[r][col].notes[num-1] = false
				cells[r][col].Refresh()
			}
		}
		for row := 0; row < 9; row++ {
			if row != r && cells[row][c].notes[num-1] {
				cells[row][c].notes[num-1] = false
				cells[row][c].Refresh()
			}
		}
		startR, startC := (r/3)*3, (c/3)*3
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				tr, tc := startR+i, startC+j
				if (tr != r || tc != c) && cells[tr][tc].notes[num-1] {
					cells[tr][tc].notes[num-1] = false
					cells[tr][tc].Refresh()
				}
			}
		}
	}

	noteMode := false
	statusBinding := binding.NewString()
	if len(os.Args) >= 2 {
		statusBinding.Set("Loaded " + os.Args[1])
	} else {
		if runtime.GOOS == "android" {
			statusBinding.Set("Empty Grid. Use UPLOAD to load a JSON string.")
		} else {
			statusBinding.Set("Empty Grid. Use IMPORT to load an image.")
		}
	}
	statusLabel := widget.NewLabelWithData(statusBinding)
	statusLabel.Wrapping = fyne.TextWrapWord

	// Solver variables
	solverStop := make(chan struct{})
	solverRunning := false
	var solverMutex sync.Mutex

	clearFailedCache := func() {
		// Cache no longer used in recursive version
	}

	goldFinger := func() {
		solverMutex.Lock()
		if solverRunning {
			solverMutex.Unlock()
			return
		}
		solverRunning = true
		solverStop = make(chan struct{})
		solverMutex.Unlock()

		// Capture state SYNCHRONOUSLY to avoid race conditions
		initialVals := [9][9]int{}
		fmt.Println("--- Gold Finger Initial Grid ---")
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				initialVals[r][c] = cells[r][c].val
				fmt.Printf("%d ", initialVals[r][c])
				if c == 2 || c == 5 {
					fmt.Print("| ")
				}
			}
			fmt.Println()
			if r == 2 || r == 5 {
				fmt.Println("------+-------+------")
			}
		}

		go func() {
			defer func() {
				solverMutex.Lock()
				solverRunning = false
				solverMutex.Unlock()
			}()

			fyne.Do(func() {
				statusBinding.Set("Gold Finger is solving...")
			})

			type boardState struct {
				vals  [9][9]int
				notes [9][9][9]bool
			}

			// Initialize state
			initialState := boardState{vals: initialVals}
			for r := 0; r < 9; r++ {
				for c := 0; c < 9; c++ {
					for n := 0; n < 9; n++ {
						initialState.notes[r][c][n] = true
					}
				}
			}

			// Validate and Propagate Initial Grid
			for r := 0; r < 9; r++ {
				for c := 0; c < 9; c++ {
					val := initialState.vals[r][c]
					if val > 0 {
						// Check for immediate conflicts
						for i := 0; i < 9; i++ {
							if i != c && initialState.vals[r][i] == val {
								fmt.Printf("CONFLICT: Row %d, Col %d and %d both have %d\n", r+1, c+1, i+1, val)
								fyne.Do(func() { statusBinding.Set(fmt.Sprintf("Conflict in Row %d", r+1)) })
								return
							}
							if i != r && initialState.vals[i][c] == val {
								fmt.Printf("CONFLICT: Col %d, Row %d and %d both have %d\n", c+1, r+1, i+1, val)
								fyne.Do(func() { statusBinding.Set(fmt.Sprintf("Conflict in Col %d", c+1)) })
								return
							}
						}
						sr, sc := (r/3)*3, (c/3)*3
						for i := 0; i < 3; i++ {
							for j := 0; j < 3; j++ {
								tr, tc := sr+i, sc+j
								if (tr != r || tc != c) && initialState.vals[tr][tc] == val {
									fmt.Printf("CONFLICT: 3x3 Block at %d,%d and %d,%d both have %d\n", r+1, c+1, tr+1, tc+1, val)
									fyne.Do(func() { statusBinding.Set("Conflict in 3x3 Block") })
									return
								}
							}
						}

						// Propagate
						for i := 0; i < 9; i++ {
							initialState.notes[r][i][val-1] = false
							initialState.notes[i][c][val-1] = false
						}
						for i := 0; i < 3; i++ {
							for j := 0; j < 3; j++ {
								initialState.notes[sr+i][sc+j][val-1] = false
							}
						}
					}
				}
			}

			// Propagation function for recursion
			var propagate func(state *boardState, r, c, val int)
			propagate = func(state *boardState, r, c, val int) {
				state.vals[r][c] = val
				for i := 0; i < 9; i++ {
					state.notes[r][i][val-1] = false
					state.notes[i][c][val-1] = false
				}
				sr, sc := (r/3)*3, (c/3)*3
				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						state.notes[sr+i][sc+j][val-1] = false
					}
				}
			}

			var solveRecursive func(state boardState) (bool, [9][9]int)
			solveRecursive = func(state boardState) (bool, [9][9]int) {
				select {
				case <-solverStop:
					return false, [9][9]int{}
				default:
				}

				// Find MRV cell
				r, c := -1, -1
				minNotes := 10
				allFilled := true

				for row := 0; row < 9; row++ {
					for col := 0; col < 9; col++ {
						if state.vals[row][col] == 0 {
							allFilled = false
							cnt := 0
							for n := 0; n < 9; n++ {
								if state.notes[row][col][n] {
									cnt++
								}
							}
							if cnt == 0 {
								return false, [9][9]int{} // Dead end
							}
							if cnt < minNotes {
								minNotes = cnt
								r, c = row, col
							}
						}
					}
				}

				if allFilled {
					return true, state.vals
				}

				// Try candidates
				for n := 0; n < 9; n++ {
					if state.notes[r][c][n] {
						nextState := state
						propagate(&nextState, r, c, n+1)
						success, result := solveRecursive(nextState)
						if success {
							return true, result
						}
					}
				}

				return false, [9][9]int{}
			}

			success, finalVals := solveRecursive(initialState)

			if success {
				fyne.Do(func() {
					for r := 0; r < 9; r++ {
						for c := 0; c < 9; c++ {
							cells[r][c].val = finalVals[r][c]
							for n := 0; n < 9; n++ {
								cells[r][c].notes[n] = false
							}
						}
					}
					bigGrid.Refresh()
					statusBinding.Set("Gold Finger Success!")
				})
			} else {
				fyne.Do(func() {
					statusBinding.Set("Gold Finger failed: No solution found.")
				})
			}
		}()
	}

	stopGoldFinger := func() {
		solverMutex.Lock()
		if solverRunning {
			close(solverStop)
		}
		solverMutex.Unlock()
		clearFailedCache()

		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if !cells[r][c].isLocked {
					cells[r][c].val = 0
					for i := 0; i < 9; i++ {
						cells[r][c].notes[i] = false
					}
				}
			}
		}
		bigGrid.Refresh()
		statusBinding.Set("Gold Finger Stopped and Board Reset.")
	}

	importImage := func() {
		clearFailedCache()
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			path := reader.URI().Path()
			statusBinding.Set("Extracting from " + filepath.Base(path) + "...")

			inputData, err := io.ReadAll(reader)
			if err != nil {
				statusBinding.Set("Import Failed: " + err.Error())
				return
			}

			go func() {
				// Copy file to local directory to bypass Gemini CLI workspace restrictions
				localPath := "import_temp" + filepath.Ext(path)
				_ = os.WriteFile(localPath, inputData, 0644)
				defer os.Remove(localPath)

				importCmd := exec.Command("gemini", localPath, "ACT AS A SUDOKU SCANNER. Extract the 9x9 grid from this image. Return ONLY a JSON object: {\"grid\": [[...],[...],...]}. Zero (0) for empty cells. CRITICAL: Every row MUST have exactly 9 numbers. NO MARKDOWN. NO CHAT.")
				output, err := importCmd.CombinedOutput()
				fmt.Printf("--- RAW GEMINI OUTPUT ---\n%s\n-------------------------\n", string(output))
				if err != nil {
					fyne.Do(func() {
						statusBinding.Set("Import Failed: " + err.Error())
					})
					return
				}

				newGrid := parseGrid(output)
				if newGrid == nil {
					fyne.Do(func() {
						statusBinding.Set("Import Failed: Could not parse Gemini output")
					})
					return
				}

				// Validate the imported grid
				var validationGrid [9][9]int
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						validationGrid[r][c] = newGrid[r][c]
					}
				}
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						if validationGrid[r][c] > 0 {
							tempVal := validationGrid[r][c]
							validationGrid[r][c] = 0 // Temporarily clear to check validity
							if !isValidMove(validationGrid, r, c, tempVal) {
								fyne.Do(func() {
									statusBinding.Set(fmt.Sprintf("Import Error: Invalid grid at %d,%d", r+1, c+1))
								})
								fmt.Printf("ERROR: AI provided invalid grid value %d at %d,%d\n", tempVal, r, c)
								return
							}
							validationGrid[r][c] = tempVal
						}
					}
				}

				newBaseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
				newSaveFileName := newBaseName + "_savegame.json"

				fyne.Do(func() {
					baseName = newBaseName
					saveFileName = newSaveFileName
					for r := 0; r < 9; r++ {
						for c := 0; c < 9; c++ {
							val := newGrid[r][c]
							cells[r][c].val = val
							cells[r][c].isLocked = (val > 0)
							for i := 0; i < 9; i++ {
								cells[r][c].notes[i] = false
							}
						}
					}
					bigGrid.Refresh()
					statusBinding.Set("Imported " + filepath.Base(path) + ". Click SAVE to persist.")
				})
			}()
		}, myWindow)

		// Set initial location to current directory to avoid C: root errors on Windows
		cwd, _ := os.Getwd()
		if l, err := storage.ListerForURI(storage.NewFileURI(cwd)); err == nil {
			fd.SetLocation(l)
		}

		fd.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png"}))
		fd.Show()
	}

	saveGame := func() {
		clearFailedCache()
		state := GameState{}
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				state.Values[r][c] = cells[r][c].val
				state.Locked[r][c] = cells[r][c].isLocked
				state.Notes[r][c] = cells[r][c].notes
			}
		}
		data, _ := json.Marshal(state)
		os.WriteFile(saveFileName, data, 0644)
		statusBinding.Set("Game Saved to " + saveFileName)
	}

	loadGame := func() {
		clearFailedCache()
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			path := reader.URI().Path()
			data, err := io.ReadAll(reader)
			if err != nil {
				statusBinding.Set("Failed to read " + filepath.Base(path))
				return
			}

			var state GameState
			if err := json.Unmarshal(data, &state); err != nil {
				statusBinding.Set("Failed to parse " + filepath.Base(path))
				return
			}

			fyne.Do(func() {
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						cells[r][c].val = state.Values[r][c]
						cells[r][c].isLocked = state.Locked[r][c]
						cells[r][c].notes = state.Notes[r][c]
					}
				}

				baseName = strings.TrimSuffix(filepath.Base(path), "_savegame.json")
				saveFileName = filepath.Base(path)

				bigGrid.Refresh()
				statusBinding.Set("Loaded " + filepath.Base(path))
			})
		}, myWindow)

		// Set initial location to current directory to avoid C: root errors on Windows
		cwd, _ := os.Getwd()
		if l, err := storage.ListerForURI(storage.NewFileURI(cwd)); err == nil {
			fd.SetLocation(l)
		}

		fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
	}

	uploadGrid := func() {
		clearFailedCache()
		entry := widget.NewMultiLineEntry()
		entry.SetPlaceHolder("Paste JSON here: {\"grid\": [[...],[...],...]}")
		// Removed non-existent SetMinVisibleRows

		d := dialog.NewCustomConfirm("Upload JSON Grid", "UPLOAD", "CANCEL", container.NewPadded(entry), func(ok bool) {
			if !ok || entry.Text == "" {
				return
			}

			var sg SudokuGrid
			if err := json.Unmarshal([]byte(entry.Text), &sg); err != nil {
				statusBinding.Set("Upload Failed: Invalid JSON format")
				return
			}

			if len(sg.Grid) != 9 {
				statusBinding.Set("Upload Failed: Grid must have 9 rows")
				return
			}

			// Validate structure and rules
			var validationGrid [9][9]int
			for r := 0; r < 9; r++ {
				if len(sg.Grid[r]) != 9 {
					statusBinding.Set(fmt.Sprintf("Upload Failed: Row %d must have 9 columns", r+1))
					return
				}
				for c := 0; c < 9; c++ {
					val := sg.Grid[r][c]
					if val < 0 || val > 9 {
						statusBinding.Set(fmt.Sprintf("Upload Failed: Invalid value %d at %d,%d", val, r+1, c+1))
						return
					}
					validationGrid[r][c] = val
				}
			}

			for r := 0; r < 9; r++ {
				for c := 0; c < 9; c++ {
					if validationGrid[r][c] > 0 {
						temp := validationGrid[r][c]
						validationGrid[r][c] = 0
						if !isValidMove(validationGrid, r, c, temp) {
							statusBinding.Set(fmt.Sprintf("Upload Failed: Rule violation at %d,%d", r+1, c+1))
							return
						}
						validationGrid[r][c] = temp
					}
				}
			}

			fyne.Do(func() {
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						val := sg.Grid[r][c]
						cells[r][c].val = val
						cells[r][c].isLocked = (val > 0)
						for i := 0; i < 9; i++ {
							cells[r][c].notes[i] = false
						}
					}
				}
				baseName = "pasted_game"
				saveFileName = "pasted_game_savegame.json"
				bigGrid.Refresh()
				statusBinding.Set("Successfully uploaded grid from pasted JSON")
			})
		}, myWindow)

		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	}

	autoNotes := func() {
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				cell := cells[r][c]
				if cell.val == 0 {
					var grid [9][9]int
					for row := 0; row < 9; row++ {
						for col := 0; col < 9; col++ {
							grid[row][col] = cells[row][col].val
						}
					}
					for num := 1; num <= 9; num++ {
						if isValidMove(grid, r, c, num) {
							cell.notes[num-1] = true
						} else {
							cell.notes[num-1] = false
						}
					}
					cell.Refresh()
				}
			}
		}
		statusBinding.Set("Automatic Pencil Notes Filled")
	}

	importBtn := widget.NewButton("IMPORT", importImage)
	uploadBtn := widget.NewButton("UPLOAD", uploadGrid)
	saveBtn := widget.NewButton("SAVE", saveGame)
	loadBtn := widget.NewButton("LOAD", loadGame)
	autoBtn := widget.NewButton("AUTO NOTES", autoNotes)
	goldFingerBtn := widget.NewButton("GOLD FINGER", goldFinger)
	resetBtn := widget.NewButton("RESET", stopGoldFinger)

	var modeBtn *widget.Button
	modeBtn = widget.NewButton("NORMAL", func() {
		noteMode = !noteMode
		if noteMode {
			modeBtn.SetText("NOTES")
			statusBinding.Set("Mode: NOTES")
		} else {
			modeBtn.SetText("NORMAL")
			statusBinding.Set("Mode: NORMAL")
		}
	})

	fileButtons := container.NewHBox()
	if runtime.GOOS != "android" {
		fileButtons.Add(importBtn)
	}
	fileButtons.Add(uploadBtn)
	fileButtons.Add(saveBtn)
	fileButtons.Add(loadBtn)
	fileButtons.Add(autoBtn)

	solverButtons := container.NewHBox(goldFingerBtn, resetBtn, modeBtn)

	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			val := initialGrid[r][c]
			cell := NewCell(r, c, val, val > 0, onSelect)
			cells[r][c] = cell
			cell.Refresh()
		}
	}

	bigGrid = container.New(layout.NewGridLayout(3))
	for br := 0; br < 3; br++ {
		for bc := 0; bc < 3; bc++ {
			subGrid := container.New(layout.NewGridLayout(3))
			for r := 0; r < 3; r++ {
				for c := 0; c < 3; c++ {
					subGrid.Add(cells[br*3+r][bc*3+c])
				}
			}
			block := container.NewPadded(subGrid)
			rect := canvas.NewRectangle(color.Transparent)
			rect.StrokeColor = theme.ForegroundColor()
			rect.StrokeWidth = 2
			bigGrid.Add(container.NewStack(rect, block))
		}
	}

	boardContainer := container.New(newSquareLayout(), bigGrid)

	var btns [10]*widget.Button
	numButtons := container.New(layout.NewGridLayout(10))
	highlightBtn := func(index int) {
		for i, b := range btns {
			if b == nil {
				continue
			}
			if i == index {
				b.Importance = widget.HighImportance
			} else {
				b.Importance = widget.MediumImportance
			}
			b.Refresh()
		}
	}

	for i := 0; i <= 9; i++ {
		num := i
		btn := widget.NewButton(strconv.Itoa(num), func() {
			highlightBtn(num)
			if selectedR == -1 {
				return
			}
			cell := cells[selectedR][selectedC]
			if cell.isLocked {
				return
			}
			if num == 0 {
				cell.val = 0
				for n := 0; n < 9; n++ {
					cell.notes[n] = false
				}
			} else {
				var grid [9][9]int
				for row := 0; row < 9; row++ {
					for col := 0; col < 9; col++ {
						grid[row][col] = cells[row][col].val
					}
				}
				if isValidMove(grid, selectedR, selectedC, num) {
					if noteMode {
						cell.val = 0
						cell.notes[num-1] = !cell.notes[num-1]
					} else {
						cell.val = num
						for n := 0; n < 9; n++ {
							cell.notes[n] = false
						}
						clearConflictingNotes(selectedR, selectedC, num)
					}
				}
			}
			cell.Refresh()
		})
		btns[i] = btn
		numButtons.Add(btn)
	}

	myWindow.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == "N" {
			noteMode = !noteMode
			highlightBtn(-1)
			if noteMode {
				modeBtn.SetText("NOTES")
				statusBinding.Set("Mode: NOTES (Press 'N' to toggle)")
			} else {
				modeBtn.SetText("NORMAL")
				statusBinding.Set("Mode: NORMAL (Press 'N' to toggle)")
			}
			return
		}
		if selectedR == -1 {
			return
		}
		cell := cells[selectedR][selectedC]

		// Navigation keys should work even on locked cells
		if k.Name == fyne.KeyLeft {
			if selectedC > 0 {
				onSelect(selectedR, selectedC-1)
			}
			return
		} else if k.Name == fyne.KeyRight {
			if selectedC < 8 {
				onSelect(selectedR, selectedC+1)
			}
			return
		} else if k.Name == fyne.KeyUp {
			if selectedR > 0 {
				onSelect(selectedR-1, selectedC)
			}
			return
		} else if k.Name == fyne.KeyDown {
			if selectedR < 8 {
				onSelect(selectedR+1, selectedC)
			}
			return
		}

		if cell.isLocked {
			return
		}
	})

	topPanel := container.NewVBox(statusLabel, fileButtons, solverButtons, numButtons)
	content := container.NewBorder(topPanel, nil, nil, nil, boardContainer)
	myWindow.SetContent(content)
	myWindow.SetFixedSize(false)
	myWindow.Resize(fyne.NewSize(550, 650))
	myWindow.ShowAndRun()
}

type squareLayout struct{}

func newSquareLayout() fyne.Layout { return &squareLayout{} }
func (l *squareLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}
func (l *squareLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	side := size.Width
	if size.Height < size.Width {
		side = size.Height
	}
	objects[0].Resize(fyne.NewSize(side, side))
	objects[0].Move(fyne.NewPos((size.Width-side)/2, (size.Height-side)/2))
}
