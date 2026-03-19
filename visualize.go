package main

import (
	"encoding/json"
	"fmt"
	"io"
	"image/color"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
	row, col           int
	val                int
	notes              [9]bool
	isLocked           bool
	onSelect           func(r, c int)
	selected           bool
	hovered            bool
	isConflicting      bool
	isDigitHighlighted bool
}

func (c *Cell) ShowConflict() {
	c.isConflicting = true
	c.Refresh()
	time.AfterFunc(time.Second, func() {
		fyne.Do(func() {
			c.isConflicting = false
			c.Refresh()
		})
	})
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
	
	// Optimization: Only update TextSize if it has changed significantly
	newMainSize := size.Height * 0.7
	if r.mainText.TextSize != newMainSize {
		r.mainText.TextSize = newMainSize
	}
	
	r.noteContainer.Resize(size)
	newNoteSize := size.Height * 0.22
	for _, t := range r.noteTexts {
		if t.TextSize != newNoteSize {
			t.TextSize = newNoteSize
		}
	}
}

func (r *cellRenderer) MinSize() fyne.Size {
	return fyne.NewSize(38, 38) // 1.5x increase from 25
}

func (r *cellRenderer) Refresh() {
	if r.cell.isConflicting {
		r.bg.FillColor = color.NRGBA{R: 255, G: 0, B: 0, A: 120} // Red background for errors
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
	} else if r.cell.selected {
		r.bg.FillColor = color.NRGBA{R: 80, G: 80, B: 200, A: 120} // Blue background for selection
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
	} else if r.cell.isDigitHighlighted {
		r.bg.FillColor = color.NRGBA{R: 39, G: 245, B: 63, A: 100} // Light green for digit highlight
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
	} else if r.cell.hovered {
		r.bg.FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30}
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
	} else {
		r.bg.FillColor = color.NRGBA{0, 0, 0, 1} // Almost transparent but solid for hit-testing
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
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
				// Highlight note if its digit is currently selected in scanning mode
				if highlightedDigit == i+1 {
					r.noteTexts[i].Color = color.NRGBA{R: 39, G: 245, B: 63, A: 255} // Light Green
					r.noteTexts[i].TextStyle = fyne.TextStyle{Bold: true}
				} else {
					r.noteTexts[i].Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255} // Standard Gray
					r.noteTexts[i].TextStyle = fyne.TextStyle{}
				}
			} else {
				r.noteTexts[i].Text = ""
			}
		}
	}
	r.bg.Refresh()
}

func (r *cellRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *cellRenderer) Destroy()                     {}

func (c *Cell) Tapped(ev *fyne.PointEvent) {
	fmt.Printf("Cell Tapped at %d,%d, InteractionMode: %v, Val: %d\n", c.row, c.col, interactionMode, c.val)
	c.onSelect(c.row, c.col)
	
	if c.val > 0 {
		// Mimic digit button behavior when no cell is selected (Toggle Scanning)
		// regardless the click mode
		num := c.val
		if highlightedDigit == num {
			updateDigitHighlights(-1)
			highlightBtn(-1)
		} else {
			updateDigitHighlights(num)
			highlightBtn(num)
		}
	} else if interactionMode && highlightedDigit > 0 && handleInput != nil {
		// If in SET mode and tapping an empty cell, place the highlighted digit
		handleInput(c.row, c.col, highlightedDigit, false)
	}
}

func (c *Cell) MouseDown(ev *desktop.MouseEvent) {
	fmt.Printf("Cell MouseDown: Button=%v at %d,%d\n", ev.Button, c.row, c.col)
	if ev.Button == desktop.MouseButtonSecondary {
		c.SecondaryTapped(&fyne.PointEvent{Position: ev.Position})
	}
}

func (c *Cell) MouseUp(ev *desktop.MouseEvent) {}

func (c *Cell) LongTapped(ev *fyne.PointEvent) {
	fmt.Printf("Cell LongTapped at %d,%d\n", c.row, c.col)
	c.SecondaryTapped(ev)
}

func (c *Cell) SecondaryTapped(ev *fyne.PointEvent) {
	fmt.Printf("Cell SecondaryTapped at %d,%d, Highlighted Digit: %d, Mode: %v\n", c.row, c.col, highlightedDigit, noteMode)
	// First select the cell
	c.onSelect(c.row, c.col)
	
	// If a digit button is selected (scanning mode), apply it using CURRENT mode
	if highlightedDigit > 0 && handleInput != nil {
		handleInput(c.row, c.col, highlightedDigit, false)
		modeStr := "Digit"
		if noteMode {
			modeStr = "Note"
		}
		if statusBinding != nil {
			statusBinding.Set(fmt.Sprintf("Placed %s %d at %d,%d", modeStr, highlightedDigit, c.row+1, c.col+1))
		}
	} else {
		if statusBinding != nil {
			statusBinding.Set("Right-click: No digit highlighted to place.")
		}
	}
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

type compactTheme struct {
	fyne.Theme
}

func (c *compactTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 30, G: 30, B: 30, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 45, G: 45, B: 48, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 28, G: 28, B: 28, A: 255}
	case theme.ColorNameForeground:
		return color.White
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 122, B: 204, A: 255} // Professional Blue
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 37, G: 37, B: 38, A: 255}
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (c *compactTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 12.5 // Tuned for high-DPI parity
	}
	if name == theme.SizeNamePadding {
		return 2
	}
	if name == theme.SizeNameInlineIcon {
		return 14
	}
	if name == theme.SizeNameScrollBar {
		return 4
	}
	return theme.DefaultTheme().Size(name)
}

type BackgroundTapper struct {
	widget.BaseWidget
	onTapped func()
}

func (b *BackgroundTapper) Tapped(_ *fyne.PointEvent) {
	b.onTapped()
}

func (b *BackgroundTapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func NewBackgroundTapper(onTapped func()) *BackgroundTapper {
	b := &BackgroundTapper{onTapped: onTapped}
	b.ExtendBaseWidget(b)
	return b
}

type SudokuBoard struct {
	widget.BaseWidget
	cells    [9][9]*Cell
	onSelect func(r, c int)
}

func NewSudokuBoard(initialGrid [][]int, onSelect func(r, c int)) *SudokuBoard {
	b := &SudokuBoard{onSelect: onSelect}
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			val := initialGrid[r][c]
			b.cells[r][c] = NewCell(r, c, val, val > 0, onSelect)
		}
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *SudokuBoard) Tapped(_ *fyne.PointEvent) {
	b.onSelect(-1, -1)
}

func (b *SudokuBoard) MouseDown(ev *desktop.MouseEvent) {
	fmt.Printf("Board MouseDown: Button=%v at pos %v\n", ev.Button, ev.Position)
	if ev.Button == desktop.MouseButtonSecondary {
		b.SecondaryTapped(&fyne.PointEvent{Position: ev.Position})
	}
}

func (b *SudokuBoard) MouseUp(ev *desktop.MouseEvent) {}

func (b *SudokuBoard) LongTapped(ev *fyne.PointEvent) {
	fmt.Printf("Board LongTapped at pos: %v\n", ev.Position)
	b.SecondaryTapped(ev)
}

func (b *SudokuBoard) SecondaryTapped(ev *fyne.PointEvent) {
	fmt.Printf("Board SecondaryTapped at pos: %v, Highlighted Digit: %d, Mode: %v\n", ev.Position, highlightedDigit, noteMode)
	
	// Calculate cell index based on click position
	size := b.Size()
	side := size.Width
	if size.Height < size.Width {
		side = size.Height
	}
	cellSize := side / 9.0
	offsetX := (size.Width - side) / 2.0
	offsetY := (size.Height - side) / 2.0
	
	col := int((ev.Position.X - offsetX) / cellSize)
	row := int((ev.Position.Y - offsetY) / cellSize)
	
	if row >= 0 && row < 9 && col >= 0 && col < 9 {
		fmt.Printf("Board-level Right-click detected for cell %d,%d\n", row, col)
		b.cells[row][col].SecondaryTapped(ev)
	}
}

func (b *SudokuBoard) CreateRenderer() fyne.WidgetRenderer {
	// Background to clear artifacts during resize
	bg := canvas.NewRectangle(color.Black)

	// 10x10 Grid Lines using Rectangles for perfect invalidation and layout scaling
	hLines := make([]*canvas.Rectangle, 10)
	vLines := make([]*canvas.Rectangle, 10)
	for i := 0; i < 10; i++ {
		h := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 30})
		v := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 30})
		// Thick lines for 3x3 blocks
		if i%3 == 0 {
			h.FillColor = color.White
			v.FillColor = color.White
		}
		hLines[i] = h
		vLines[i] = v
	}

	var objects []fyne.CanvasObject
	objects = append(objects, bg) // Background MUST be first
	for _, l := range hLines {
		objects = append(objects, l)
	}
	for _, l := range vLines {
		objects = append(objects, l)
	}
	// Cells MUST be last to be on top and receive all tap/click events
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			objects = append(objects, b.cells[r][c])
		}
	}

	return &boardRenderer{
		board:   b,
		bg:      bg,
		hLines:  hLines,
		vLines:  vLines,
		objects: objects,
	}
}

type boardRenderer struct {
	board   *SudokuBoard
	bg      *canvas.Rectangle
	hLines  []*canvas.Rectangle
	vLines  []*canvas.Rectangle
	objects []fyne.CanvasObject

	// Debouncing fields
	lastSize    fyne.Size
	resizeTimer *time.Timer
	mu          sync.Mutex
}

func (r *boardRenderer) Layout(size fyne.Size) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Instant feedback: Resize the background immediately so it doesn't look blank
	side := size.Width
	if size.Height < size.Width {
		side = size.Height
	}
	offsetX := (size.Width - side) / 2.0
	offsetY := (size.Height - side) / 2.0
	r.bg.Resize(fyne.NewSize(side, side))
	r.bg.Move(fyne.NewPos(offsetX, offsetY))

	// 2. If the size is still changing, stop the previous timer
	if r.resizeTimer != nil {
		r.resizeTimer.Stop()
	}

	// 3. Render immediately ONLY if the change is tiny (to avoid jitter)
	// Otherwise, debounce the heavy cell/line layout
	r.resizeTimer = time.AfterFunc(100*time.Millisecond, func() {
		fyne.Do(func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.performLayout(size)
		})
	})
}

func (r *boardRenderer) performLayout(size fyne.Size) {
	side := size.Width
	if size.Height < size.Width {
		side = size.Height
	}
	
	// Float32 for perfect pixel alignment
	cellSize := side / 9.0 
	offsetX := (size.Width - side) / 2.0
	offsetY := (size.Height - side) / 2.0

	// Move and Resize cells
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			c := r.board.cells[row][col]
			c.Resize(fyne.NewSize(cellSize, cellSize))
			c.Move(fyne.NewPos(offsetX+(float32(col)*cellSize), offsetY+(float32(row)*cellSize)))
		}
	}

	// Move and Resize grid lines
	for i := 0; i < 10; i++ {
		pos := float32(i) * cellSize
		thickness := float32(1.0)
		if i%3 == 0 {
			thickness = 2.0
		}
		
		// Horizontal lines
		r.hLines[i].Resize(fyne.NewSize(side, thickness))
		r.hLines[i].Move(fyne.NewPos(offsetX, offsetY+pos-(thickness/2.0)))
		
		// Vertical lines
		r.vLines[i].Resize(fyne.NewSize(thickness, side))
		r.vLines[i].Move(fyne.NewPos(offsetX+pos-(thickness/2.0), offsetY))
	}
	
	r.board.Refresh()
}

func (r *boardRenderer) MinSize() fyne.Size {
	return fyne.NewSize(338, 338) // 225 * 1.5
}

func (r *boardRenderer) Refresh() {
	// 1. Refresh the background and lines
	for i := 0; i < 10; i++ {
		if i%3 == 0 {
			r.hLines[i].FillColor = color.White
			r.vLines[i].FillColor = color.White
		} else {
			r.hLines[i].FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30}
			r.vLines[i].FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30}
		}
		r.hLines[i].Refresh()
		r.vLines[i].Refresh()
	}
	r.bg.FillColor = theme.BackgroundColor()
	r.bg.Refresh()

	// 2. CRITICAL FIX: Explicitly refresh all 81 cells to reflect state changes (like RESET)
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			r.board.cells[row][col].Refresh()
		}
	}

	canvas.Refresh(r.board)
}

func (r *boardRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *boardRenderer) Destroy()                     {}

var (
	btns                  [10]*widget.Button
	updateButtonStates    func()
	highlightBtn          func(index int)
	board                 *SudokuBoard
	selectedR, selectedC  int = -1, -1
	highlightedDigit      int = -1
	noteMode              bool
	onSelect              func(r, c int)
	updateDigitHighlights func(num int)
	getConflictingCells   func(grid [9][9]int, r, c, num int) [][2]int
	clearConflictingNotes func(r, c, num int)
	handleInput           func(r, c, num int, useOppositeMode bool)
	checkSolved           func() bool
	stopTimer             func()
	startTimer            func()
	statusBinding         binding.String
	timerBinding          binding.String
	interactionMode       bool // false = SELECT, true = SET
)

func main() {
	fmt.Println("Sudoku Helper starting...")

	// Normalize scaling: Let Fyne handle DPI naturally by NOT forcing FYNE_SCALE
	// unless specifically requested via the --scale flag.
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--scale=") {
			scale := strings.TrimPrefix(arg, "--scale=")
			os.Setenv("FYNE_SCALE", scale)
			fmt.Printf("Manually applied UI Scale: %s\n", scale)
		}
	}
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

	updateDigitHighlights = func(num int) {
		highlightedDigit = num
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				cell := board.cells[r][c]
				cell.isDigitHighlighted = (num > 0 && cell.val == num)
				cell.Refresh()
			}
		}
	}

	onSelect = func(r, c int) {
		if selectedR != -1 && selectedR < 9 && selectedC < 9 {
			board.cells[selectedR][selectedC].selected = false
			board.cells[selectedR][selectedC].Refresh()
		}
		selectedR, selectedC = r, c
		if r != -1 && c != -1 && r < 9 && c < 9 {
			board.cells[r][c].selected = true
			board.cells[r][c].Refresh()
			// Don't clear highlights when selecting a cell
		} else {
			updateDigitHighlights(-1) // Clear digit highlights if clicking outside
			highlightBtn(-1)          // Also clear digit button highlights
		}
	}

	board = NewSudokuBoard(initialGrid, onSelect)

	getConflictingCells = func(grid [9][9]int, r, c, num int) [][2]int {
		conflicts := [][2]int{}
		// Check row
		for col := 0; col < 9; col++ {
			if col != c && grid[r][col] == num {
				conflicts = append(conflicts, [2]int{r, col})
			}
		}
		// Check column
		for row := 0; row < 9; row++ {
			if row != r && grid[row][c] == num {
				conflicts = append(conflicts, [2]int{row, c})
			}
		}
		// Check 3x3 box
		startR, startC := (r/3)*3, (c/3)*3
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				tr, tc := startR+i, startC+j
				if (tr != r || tc != c) && grid[tr][tc] == num {
					conflicts = append(conflicts, [2]int{tr, tc})
				}
			}
		}
		return conflicts
	}

	isValidMove := func(grid [9][9]int, r, c, num int) bool {
		return len(getConflictingCells(grid, r, c, num)) == 0
	}

	clearConflictingNotes = func(r, c, num int) {
		for col := 0; col < 9; col++ {
			if col != c && board.cells[r][col].notes[num-1] {
				board.cells[r][col].notes[num-1] = false
				board.cells[r][col].Refresh()
			}
		}
		for row := 0; row < 9; row++ {
			if row != r && board.cells[row][c].notes[num-1] {
				board.cells[row][c].notes[num-1] = false
				board.cells[row][c].Refresh()
			}
		}
		startR, startC := (r/3)*3, (c/3)*3
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				tr, tc := startR+i, startC+j
				if (tr != r || tc != c) && board.cells[tr][tc].notes[num-1] {
					board.cells[tr][tc].notes[num-1] = false
					board.cells[tr][tc].Refresh()
				}
			}
		}
	}

	noteMode = false
	statusBinding = binding.NewString()
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

	// Timer variables
	var startTime time.Time
	var totalElapsed time.Duration
	timerRunning := false
	timerBinding = binding.NewString()
	timerBinding.Set("00:00:00")


	updateTimerDisplay := func() {
		dur := totalElapsed
		if timerRunning {
			dur += time.Since(startTime)
		}
		h := int(dur.Hours())
		m := int(dur.Minutes()) % 60
		s := int(dur.Seconds()) % 60
		timerBinding.Set(fmt.Sprintf("%02d:%02d:%02d", h, m, s))
	}

	go func() {
		for {
			time.Sleep(250 * time.Millisecond) // Higher frequency for better responsiveness
			if timerRunning {
				fyne.Do(updateTimerDisplay)
			}
		}
	}()

	startTimer = func() {
		if !timerRunning {
			startTime = time.Now()
			timerRunning = true
			fyne.Do(updateTimerDisplay) // Update immediately
		}
	}

	pauseTimer := func() {
		if timerRunning {
			totalElapsed += time.Since(startTime)
			timerRunning = false
			fyne.Do(updateTimerDisplay)
		}
	}

	stopTimer = func() {
		if timerRunning {
			totalElapsed += time.Since(startTime)
			timerRunning = false
		}
		fyne.Do(updateTimerDisplay)
	}

	checkSolved = func() bool {
		var grid [9][9]int
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if board.cells[r][c].val == 0 {
					return false
				}
				grid[r][c] = board.cells[r][c].val
			}
		}

		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				val := grid[r][c]
				grid[r][c] = 0
				if !isValidMove(grid, r, c, val) {
					return false
				}
				grid[r][c] = val
			}
		}
		return true
	}

	handleInput = func(r, c, num int, useOppositeMode bool) {
		if r == -1 || c == -1 || r >= 9 || c >= 9 {
			return
		}
		cell := board.cells[r][c]
		if cell.isLocked {
			return
		}

		actualNoteMode := noteMode
		if useOppositeMode {
			actualNoteMode = !noteMode
		}

		startTimer() // Start timer on input

		if num == 0 {
			cell.val = 0
			for n := 0; n < 9; n++ {
				cell.notes[n] = false
			}
			// Refresh highlights to account for the removed digit, if a highlight is active
			if highlightedDigit != -1 {
				updateDigitHighlights(highlightedDigit)
			}
		} else {
			var grid [9][9]int
			for row := 0; row < 9; row++ {
				for col := 0; col < 9; col++ {
					grid[row][col] = board.cells[row][col].val
				}
			}
			conflicts := getConflictingCells(grid, r, c, num)
			if len(conflicts) == 0 {
				if actualNoteMode {
					cell.val = 0
					cell.notes[num-1] = !cell.notes[num-1]
				} else {
					cell.val = num
					for n := 0; n < 9; n++ {
						cell.notes[n] = false
					}
					clearConflictingNotes(r, c, num)
					updateDigitHighlights(num)
					highlightBtn(num)
				}
			} else {
				for _, coord := range conflicts {
					board.cells[coord[0]][coord[1]].ShowConflict()
				}
			}
		}
		cell.Refresh()
		updateButtonStates()
		if checkSolved() {
			stopTimer()
			val, _ := timerBinding.Get()
			statusBinding.Set("Puzzle Solved! Final Time: " + val)
		}
	}

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
				initialVals[r][c] = board.cells[r][c].val
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
							board.cells[r][c].val = finalVals[r][c]
							for n := 0; n < 9; n++ {
								board.cells[r][c].notes[n] = false
							}
						}
					}
					board.Refresh()
					updateButtonStates()
					updateDigitHighlights(-1)
					highlightBtn(-1)
					stopTimer()
					val, _ := timerBinding.Get()
					statusBinding.Set("Gold Finger Success! Time: " + val)
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

		// Reset timer
		timerRunning = false
		totalElapsed = 0
		fyne.Do(updateTimerDisplay)

		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if !board.cells[r][c].isLocked {
					board.cells[r][c].val = 0
					for i := 0; i < 9; i++ {
						board.cells[r][c].notes[i] = false
					}
				}
			}
		}
		board.Refresh()
		updateButtonStates()
		updateDigitHighlights(-1)
		highlightBtn(-1)
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
							board.cells[r][c].val = val
							board.cells[r][c].isLocked = (val > 0)
							for i := 0; i < 9; i++ {
								board.cells[r][c].notes[i] = false
							}
						}
					}
					board.Refresh()
					updateButtonStates()
					updateDigitHighlights(-1)
					highlightBtn(-1)
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
				state.Values[r][c] = board.cells[r][c].val
				state.Locked[r][c] = board.cells[r][c].isLocked
				state.Notes[r][c] = board.cells[r][c].notes
			}
		}
		data, _ := json.Marshal(state)

		handleWriter := func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}

			// Single-pass write to ensure atomic-like behavior on SAF
			_, writeErr := writer.Write(data)
			closeErr := writer.Close()

			if writeErr != nil {
				statusBinding.Set("Failed to write data: " + writeErr.Error())
				return
			}
			if closeErr != nil {
				statusBinding.Set("Failed to finalize save: " + closeErr.Error())
				return
			}

			// Fix for Android: unescape the URI name to get a human-readable string
			rawName := writer.URI().Name()
			if unescaped, err := url.QueryUnescape(rawName); err == nil {
				if strings.HasSuffix(unescaped, ".json") {
					saveFileName = unescaped
					baseName = strings.TrimSuffix(saveFileName, "_savegame.json")
				}
			}

			statusBinding.Set("Game Saved to " + saveFileName)
		}

		if runtime.GOOS == "android" || runtime.GOOS == "ios" {
			// On Android, 'NewFileSave' can be unresponsive for overwriting existing files.
			// We provide a dedicated 'OVERWRITE' choice that uses 'NewFileOpen' (which is responsive)
			// to pick the target file, and then we obtain a writer for it.
			d := dialog.NewCustomConfirm("Save Game", "OVERWRITE", "NEW FILE",
				widget.NewLabel("Choose 'OVERWRITE' to pick an existing file,\nor 'NEW FILE' to create a new one."),
				func(isOverwrite bool) {
					if isOverwrite {
						fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
							if err != nil || reader == nil {
								return
							}
							uri := reader.URI()
							reader.Close() // Close reader to free up the file for writing
							
							writer, err := storage.Writer(uri)
							if err != nil {
								statusBinding.Set("Error opening for write: " + err.Error())
								return
							}
							handleWriter(writer, nil)
						}, myWindow)
						fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
						fd.Show()
					} else {
						fd := dialog.NewFileSave(handleWriter, myWindow)
						fd.SetFileName(saveFileName)
						fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
						fd.Show()
					}
				}, myWindow)
			d.Show()
		} else {
			// Desktop: standard Save dialog with pre-filled name
			fd := dialog.NewFileSave(handleWriter, myWindow)
			cwd, _ := os.Getwd()
			if l, err := storage.ListerForURI(storage.NewFileURI(cwd)); err == nil {
				fd.SetLocation(l)
			}
			fd.SetFileName(saveFileName)
			fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
			fd.Show()
		}
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
				statusBinding.Set("Failed to parse " + reader.URI().Name())
				return
			}

			fyne.Do(func() {
				for r := 0; r < 9; r++ {
					for c := 0; c < 9; c++ {
						board.cells[r][c].val = state.Values[r][c]
						board.cells[r][c].isLocked = state.Locked[r][c]
						board.cells[r][c].notes = state.Notes[r][c]
					}
				}

				// Fix for Android: Use url.QueryUnescape to get the human-readable name
				rawName := reader.URI().Name()
				if unescaped, err := url.QueryUnescape(rawName); err == nil {
					if strings.HasSuffix(unescaped, ".json") {
						saveFileName = unescaped
						baseName = strings.TrimSuffix(saveFileName, "_savegame.json")
					}
				}

				board.Refresh()
				updateButtonStates()
				updateDigitHighlights(-1)
				highlightBtn(-1)
				statusBinding.Set("Loaded " + saveFileName)
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
						board.cells[r][c].val = val
						board.cells[r][c].isLocked = (val > 0)
						for i := 0; i < 9; i++ {
							board.cells[r][c].notes[i] = false
						}
					}
				}
				baseName = "pasted_game"
				saveFileName = "pasted_game_savegame.json"
				board.Refresh()
				updateButtonStates()
				updateDigitHighlights(-1)
				highlightBtn(-1)
				statusBinding.Set("Successfully uploaded grid from pasted JSON")
			})
		}, myWindow)

		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	}

	autoNotes := func() {
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				cell := board.cells[r][c]
				if cell.val == 0 {
					var grid [9][9]int
					for row := 0; row < 9; row++ {
						for col := 0; col < 9; col++ {
							grid[row][col] = board.cells[row][col].val
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

	timerLabel := widget.NewLabelWithData(timerBinding)
	pauseBtn := widget.NewButton("PAUSE", pauseTimer)

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

	controlsRow := container.NewHBox(goldFingerBtn, resetBtn, modeBtn, pauseBtn)

	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			val := initialGrid[r][c]
			cell := NewCell(r, c, val, val > 0, onSelect)
			board.cells[r][c] = cell
			cell.Refresh()
		}
	}

	numButtons := container.New(layout.NewGridLayout(5))
	updateButtonStates = func() {
		counts := make(map[int]int)
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if board.cells[r][c].val > 0 {
					counts[board.cells[r][c].val]++
				}
			}
		}
		for i := 1; i <= 9; i++ {
			if btns[i] != nil {
				if counts[i] >= 9 {
					btns[i].Disable()
				} else {
					btns[i].Enable()
				}
			}
		}
	}

	highlightBtn = func(index int) {
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
			if selectedR == -1 {
				highlightBtn(num)
				if highlightedDigit == num {
					updateDigitHighlights(-1)
					highlightBtn(-1)
				} else {
					updateDigitHighlights(num)
				}
				return
			}
			handleInput(selectedR, selectedC, num, false)
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

		num := -1
		if k.Name >= "0" && k.Name <= "9" {
			num, _ = strconv.Atoi(string(k.Name))
		} else if k.Name == fyne.KeyBackspace || k.Name == fyne.KeyDelete {
			num = 0
		}

		// Digit highlight mode when no cell is selected
		if selectedR == -1 {
			if num > 0 {
				if highlightedDigit == num {
					updateDigitHighlights(-1)
					highlightBtn(-1)
				} else {
					updateDigitHighlights(num)
					highlightBtn(num)
				}
			}
			return
		}
		
		// If a cell IS selected, handle navigation and editing
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

		cell := board.cells[selectedR][selectedC]
		if cell.isLocked {
			return
		}

		if num >= 0 && num <= 9 {
			handleInput(selectedR, selectedC, num, false)
		}
	})

	interactionToggle := widget.NewRadioGroup([]string{"SELECT", "SET"}, func(val string) {
		interactionMode = (val == "SET")
		statusBinding.Set("Interaction Mode: " + val)
	})
	interactionToggle.Horizontal = true
	interactionToggle.SetSelected("SELECT")
	interactionRow := container.NewHBox(widget.NewLabel("CLICK MODE:"), interactionToggle)

	statusRow := container.NewBorder(nil, nil, nil, timerLabel, statusLabel)
	topPanel := container.NewVBox(statusRow, fileButtons, controlsRow, numButtons, interactionRow)
	// Apply compact theme to the top control panel
	compact := &compactTheme{Theme: theme.DefaultTheme()}
	topPanelWithTheme := container.NewThemeOverride(topPanel, compact)

	content := container.NewBorder(topPanelWithTheme, nil, nil, nil, board)
	
	// Wrap everything in a stack with a background tapper to clear selection when clicking outside
	mainContent := container.NewStack(
		NewBackgroundTapper(func() { onSelect(-1, -1) }),
		content,
	)
	myWindow.SetContent(mainContent)

	if runtime.GOOS != "android" {
		myWindow.SetFixedSize(false)
	} else {
		myWindow.SetFixedSize(true)
	}

	myWindow.Resize(fyne.NewSize(500, 750))
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
