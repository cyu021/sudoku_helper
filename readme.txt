# Sudoku Helper

A Golang toolkit to extract Sudoku grids from screenshots and solve them via a feature-rich, interactive GUI.

## Features
- **AI Extraction**: Uses Gemini CLI to convert Sudoku screenshots into digital grids.
- **Interactive GUI**: Built with Fyne, featuring a responsive, touch-friendly layout.
- **Gold Finger Solver**: A learning-based AI solver with a contextual failure cache and 100k restart limit.
- **Multi-Platform Support**:
  - Linux (x86_64 and ARM64)
  - Windows (Statically linked, console-less)
  - Android (ARM64 APK)
- **Automatic Notes**: Intelligent pencil-mark management based on Sudoku rules.
- **Embedded Assets**: Standalone execution with embedded icons.

## Requirements
- Go 1.21+
- Fyne-cross (for multi-platform builds)
- Gemini CLI (for image extraction)

## Build Instructions
- **Linux Native**: `go build -o sudoku_helper_linux visualize.go bundled.go`
- **Windows**: `fyne-cross windows -arch=amd64 .`
- **Android**: `fyne-cross android -arch=arm64 .`
- **ARM64 Linux**: `fyne-cross linux -arch=arm64 .`

## Usage
1. Launch the application.
2. Use **IMPORT** to load a grid from a screenshot (requires Gemini CLI) or manual entry.
3. Use **AUTO NOTES** to generate pencil marks.
4. Click **GOLD FINGER** to trigger the AI solver for difficult puzzles.
5. Use **SAVE/LOAD** to manage puzzle progress.

## License
MIT License
