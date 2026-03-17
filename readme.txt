# Sudoku Helper

A Golang toolkit to extract Sudoku grids from screenshots and solve them via a feature-rich, interactive GUI.

## Features
- **AI Extraction**: Uses Gemini CLI to convert Sudoku screenshots into digital grids.
- **Interactive GUI**: Built with Fyne, featuring a responsive, touch-friendly layout.
- **Recursive MRV Solver**: A deterministic, instant AI solver using the Minimum Remaining Values (MRV) heuristic.
- **Multi-Platform Support**:
  - Linux (x86_64 and ARM64)
  - Windows (Statically linked, console-less)
  - Android (ARM64 APK)
- **Automatic Notes**: Intelligent pencil-mark management based on Sudoku rules.
- **Embedded Assets**: Standalone execution with embedded icons.

## Requirements
- Go 1.21+
- Fyne-cross (for multi-platform/Android builds)
- Gemini CLI v0.33.2+ (for high-accuracy image extraction with Gemini 3 models)

## Build Instructions (via WSL/Linux)
### Linux (Native/WSL)
```bash
NPROC=$(nproc); GOMAX=$((NPROC > 2 ? NPROC - 2 : 1)); go build -v -p $GOMAX -o sudoku_helper_linux visualize.go bundled.go
```

### Windows (Cross-compile via Mingw-w64)
```bash
NPROC=$(nproc); GOMAX=$((NPROC > 2 ? NPROC - 2 : 1)); GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -v -p $GOMAX -ldflags="-s -w -extldflags=-static -H=windowsgui" -o sudoku_helper.exe visualize.go bundled.go
```

### ARM64 Linux (via Fyne-cross)
```bash
fyne-cross linux --arch=arm64 --icon=Icon.png --name=sudoku_helper_arm64_static --app-id=com.gemini.sudoku --ldflags="-extldflags=-static" .
```

### Android (via Fyne-cross)
```bash
fyne-cross android --arch=arm64 --icon=Icon.png --name=SudokuHelper --app-id=com.gemini.sudoku .
```

## Usage
1. Launch the application.
2. Use **IMPORT** to load a grid from a screenshot (requires Gemini CLI) or manual entry.
3. Use **AUTO NOTES** to generate pencil marks.
4. Click **GOLD FINGER** to trigger the instant recursive solver.
5. Use **SAVE/LOAD** to manage puzzle progress.

## Troubleshooting
- **Windows File Picker Logs:** You may see "Error getting file attributes" in the console when browsing the `C:` root. These are harmless library logs from Fyne and do not affect functionality.
- **Gemini Extraction:** Ensure your Gemini CLI is logged in and has access to Gemini 3 models for the best results.

## License
MIT License
