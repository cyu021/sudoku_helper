package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SudokuGrid struct {
	Grid [][]int `json:"grid"`
}

type GameState struct {
	Values [][]int     `json:"values"`
	Locked [][]bool    `json:"locked"`
	Notes  [][][9]bool `json:"notes"`
}

type Cell struct {
	widget.BaseWidget
	row, col int
	val      int
	notes    [9]bool
	isLocked bool
	onSelect func(r, c int)
	selected bool
}

func NewCell(r, c, val int, isLocked bool, onSelect func(r, c int)) *Cell {
	cell := &Cell{row: r, col: c, val: val, isLocked: isLocked, onSelect: onSelect}
	cell.ExtendBaseWidget(cell)
	return cell
}

func (c *Cell) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 40}
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
	r.mainText.TextSize = size.Height * 0.6
	r.noteContainer.Resize(size)
	noteSize := size.Height * 0.2
	for _, t := range r.noteTexts {
		t.TextSize = noteSize
	}
}

func (r *cellRenderer) MinSize() fyne.Size {
	return fyne.NewSize(20, 20)
}

func (r *cellRenderer) Refresh() {
	if r.cell.selected {
		r.bg.FillColor = color.NRGBA{R: 100, G: 100, B: 255, A: 60}
	} else {
		r.bg.FillColor = color.Transparent
	}

	if r.cell.val > 0 {
		r.mainText.Text = strconv.Itoa(r.cell.val)
		if r.cell.isLocked {
			r.mainText.Color = color.NRGBA{R: 255, G: 215, B: 0, A: 255}
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
}

func (r *cellRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *cellRenderer) Destroy()                     {}

func (c *Cell) Tapped(_ *fyne.PointEvent) {
	c.onSelect(c.row, c.col)
}

func main() {
	var initialGrid [][]int
	var baseName string
	var saveFileName string

	parseGrid := func(raw []byte) [][]int {
		jsonStr := string(raw)
		gridIdx := strings.LastIndex(jsonStr, "\"grid\"")
		if gridIdx == -1 {
			return nil
		}
		start := strings.LastIndex(jsonStr[:gridIdx], "{")
		end := strings.Index(jsonStr[gridIdx:], "}")
		if start == -1 || end == -1 {
			return nil
		}
		jsonStr = jsonStr[start : gridIdx+end+1]
		var sg SudokuGrid
		if err := json.Unmarshal([]byte(jsonStr), &sg); err != nil {
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
		statusBinding.Set("Empty Grid. Use IMPORT to load an image.")
	}
	statusLabel := widget.NewLabelWithData(statusBinding)
	statusLabel.Wrapping = fyne.TextWrapWord

	// Solver variables
	failedNumbers := make(map[string][]int)
	solverStop := make(chan struct{})
	solverRunning := false
	var solverMutex sync.Mutex

	clearFailedCache := func() {
		failedNumbers = make(map[string][]int)
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

			applyAutoNotes := func(state *boardState) {
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						if state.vals[r][c] == 0 {
							for n := 0; n < 9; n++ {
								state.notes[r][c][n] = isValidMove(state.vals, r, c, n+1)
							}
						}
					}
				}
			}

			checkFailure := func(state *boardState) (bool, bool) {
				for r := 0; r < 9; r++ {
					seen := make(map[int]bool)
					for c := 0; c < 9; c++ {
						if state.vals[r][c] == 0 {
							cnt, last := 0, -1
							for n := 0; n < 9; n++ {
								if state.notes[r][c][n] {
									cnt++
									last = n
								}
							}
							if cnt == 1 {
								if seen[last] {
									return true, false
								}
								seen[last] = true
							}
						}
					}
				}
				for c := 0; c < 9; c++ {
					seen := make(map[int]bool)
					for r := 0; r < 9; r++ {
						if state.vals[r][c] == 0 {
							cnt, last := 0, -1
							for n := 0; n < 9; n++ {
								if state.notes[r][c][n] {
									cnt++
									last = n
								}
							}
							if cnt == 1 {
								if seen[last] {
									return true, false
								}
								seen[last] = true
							}
						}
					}
				}
				for br := 0; br < 3; br++ {
					for bc := 0; bc < 3; bc++ {
						seen := make(map[int]bool)
						for i := 0; i < 3; i++ {
							for j := 0; j < 3; j++ {
								r, c := br*3+i, bc*3+j
								if state.vals[r][c] == 0 {
									cnt, last := 0, -1
									for n := 0; n < 9; n++ {
										if state.notes[r][c][n] {
											cnt++
											last = n
										}
									}
									if cnt == 1 {
										if seen[last] {
											return true, false
										}
										seen[last] = true
									}
								}
							}
						}
					}
				}
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						if state.vals[r][c] == 0 {
							cnt := 0
							for n := 0; n < 9; n++ {
								if state.notes[r][c][n] {
									cnt++
								}
							}
							if cnt == 0 {
								return false, true
							}
						}
					}
				}
				return false, false
			}

			initialVals := [9][9]int{}
			for r := 0; r < 9; r++ {
				for c := 0; c < 9; c++ {
					if cells[r][c].isLocked {
						initialVals[r][c] = cells[r][c].val
					}
				}
			}

			restarts := 0
			rand.Seed(time.Now().UnixNano())

			for restarts < 100000 {
				select {
				case <-solverStop:
					fyne.Do(func() {
						statusBinding.Set("Gold Finger stopped.")
					})
					return
				default:
				}

				state := boardState{vals: initialVals}
				applyAutoNotes(&state)
				history := []string{}

				for {
					allFilled := true
					for r := 0; r < 9; r++ {
						for c := 0; c < 9; c++ {
							if state.vals[r][c] == 0 {
								allFilled = false
								break
							}
						}
					}
					if allFilled {
						finalVals := state.vals
						finalRestarts := restarts
						fyne.Do(func() {
							for r := 0; r < 9; r++ {
								for c := 0; c < 9; c++ {
									cells[r][c].val = finalVals[r][c]
									cells[r][c].notes = [9]bool{}
								}
							}
							bigGrid.Refresh()
							statusBinding.Set(fmt.Sprintf("Gold Finger Success! Restarts: %d", finalRestarts))
						})
						clearFailedCache()
						return
					}

					// (1) Pick block with most known numbers
					bestBlockR, bestBlockC := -1, -1
					maxKnown := -1
					for br := 0; br < 3; br++ {
						for bc := 0; bc < 3; bc++ {
							known := 0
							for i := 0; i < 3; i++ {
								for j := 0; j < 3; j++ {
									if state.vals[br*3+i][bc*3+j] != 0 {
										known++
									}
								}
							}
							if known < 9 && known > maxKnown {
								maxKnown = known
								bestBlockR, bestBlockC = br, bc
							}
						}
					}

					// (2) Pick grid with most pencil notes in that block
					r, c := -1, -1
					maxNotes := -1
					for i := 0; i < 3; i++ {
						for j := 0; j < 3; j++ {
							tr, tc := bestBlockR*3+i, bestBlockC*3+j
							if state.vals[tr][tc] == 0 {
								cnt := 0
								for n := 0; n < 9; n++ {
									if state.notes[tr][tc][n] {
										cnt++
									}
								}
								if cnt > maxNotes {
									maxNotes = cnt
									r, c = tr, tc
								}
							}
						}
					}

					ctx := strings.Join(history, "|")
					key := fmt.Sprintf("%s|%d,%d", ctx, r, c)

					var options []int
					for n := 0; n < 9; n++ {
						if state.notes[r][c][n] {
							isFailed := false
							for _, fn := range failedNumbers[key] {
								if fn == n+1 {
									isFailed = true
									break
								}
							}
							if !isFailed {
								options = append(options, n+1)
							}
						}
					}

					if len(options) == 0 {
						if len(history) > 0 {
							lastCtx := strings.Join(history[:len(history)-1], "|")
							parts := strings.Split(history[len(history)-1], ":")
							failKey := lastCtx + "|" + parts[0]
							lastVal, _ := strconv.Atoi(parts[1])
							failedNumbers[failKey] = append(failedNumbers[failKey], lastVal)
						}
						break
					}

					val := options[rand.Intn(len(options))]
					state.vals[r][c] = val
					state.notes[r][c] = [9]bool{}
					history = append(history, fmt.Sprintf("%d,%d:%d", r, c, val))

					// Propagation
					for i := 0; i < 9; i++ {
						if i != c {
							state.notes[r][i][val-1] = false
						}
						if i != r {
							state.notes[i][c][val-1] = false
						}
					}
					sr, sc := (r/3)*3, (c/3)*3
					for i := 0; i < 3; i++ {
						for j := 0; j < 3; j++ {
							if sr+i != r || sc+j != c {
								state.notes[sr+i][sc+j][val-1] = false
							}
						}
					}

					f42, f43 := checkFailure(&state)
					if f42 || f43 {
						failedNumbers[key] = append(failedNumbers[key], val)
						break
					}
				}
				restarts++
				if restarts%1000 == 0 {
					currentRestarts := restarts
					fyne.Do(func() {
						statusBinding.Set(fmt.Sprintf("Gold Finger Solving... (%d restarts)", currentRestarts))
					})
				}
			}
			fyne.Do(func() {
				statusBinding.Set("Gold Finger failed.")
			})
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

			go func() {
				importCmd := exec.Command("gemini", path, "Extract the 9x9 Sudoku grid from this image. Return ONLY a 2D array of integers in a JSON object with a 'grid' field, where 0 represents an empty cell. Do not include markdown code blocks or extra text.")
				output, err := importCmd.CombinedOutput()
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
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png"}))
		fd.Show()
	}

	saveGame := func() {
		clearFailedCache()
		state := GameState{
			Values: make([][]int, 9),
			Locked: make([][]bool, 9),
			Notes:  make([][][9]bool, 9),
		}
		for r := 0; r < 9; r++ {
			state.Values[r] = make([]int, 9)
			state.Locked[r] = make([]bool, 9)
			state.Notes[r] = make([][9]bool, 9)
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
			data, err := os.ReadFile(path)
			if err != nil {
				statusBinding.Set("Failed to read " + filepath.Base(path))
				return
			}

			var state GameState
			if err := json.Unmarshal(data, &state); err != nil {
				statusBinding.Set("Failed to parse " + filepath.Base(path))
				return
			}

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
		}, myWindow)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
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
	saveBtn := widget.NewButton("SAVE", saveGame)
	loadBtn := widget.NewButton("LOAD", loadGame)
	autoBtn := widget.NewButton("AUTO NOTES", autoNotes)
	goldFingerBtn := widget.NewButton("GOLD FINGER", goldFinger)
	resetBtn := widget.NewButton("RESET", stopGoldFinger)

	fileButtons := container.NewHBox(importBtn, saveBtn, loadBtn, autoBtn)
	solverButtons := container.NewHBox(goldFingerBtn, resetBtn)

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
				statusBinding.Set("Mode: NOTES (Press 'N' to toggle)")
			} else {
				statusBinding.Set("Mode: NORMAL (Press 'N' to toggle)")
			}
			return
		}
		if selectedR == -1 {
			return
		}
		cell := cells[selectedR][selectedC]
		if cell.isLocked {
			return
		}
		if k.Name >= "0" && k.Name <= "9" {
			num, _ := strconv.Atoi(string(k.Name))
			highlightBtn(num)
			if num == 0 {
				cell.val = 0
				for i := 0; i < 9; i++ {
					cell.notes[i] = false
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
						for i := 0; i < 9; i++ {
							cell.notes[i] = false
						}
						clearConflictingNotes(selectedR, selectedC, num)
					}
				}
			}
			cell.Refresh()
		} else if k.Name == fyne.KeyBackspace || k.Name == fyne.KeyDelete {
			highlightBtn(0)
			cell.val = 0
			for i := 0; i < 9; i++ {
				cell.notes[i] = false
			}
			cell.Refresh()
		} else if k.Name == fyne.KeyLeft {
			if selectedC > 0 {
				onSelect(selectedR, selectedC-1)
			}
		} else if k.Name == fyne.KeyRight {
			if selectedC < 8 {
				onSelect(selectedR, selectedC+1)
			}
		} else if k.Name == fyne.KeyUp {
			if selectedR > 0 {
				onSelect(selectedR-1, selectedC)
			}
		} else if k.Name == fyne.KeyDown {
			if selectedR < 8 {
				onSelect(selectedR+1, selectedC)
			}
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
