# Sudoku Helper

A high-performance Golang toolkit for extracting Sudoku grids from screenshots and solving them via a feature-rich, interactive GUI. Built for Linux, Windows, and Android using the Fyne framework.

## Key Features
- **AI Extraction**: Uses Gemini CLI (Gemini 3 models) to convert Sudoku screenshots into digital grids with high accuracy.
- **Interactive GUI**: Built with Fyne, featuring a responsive, touch-friendly layout with hover effects and borderless selection.
- **High-Performance Grid**: Employs a 20-rectangle architecture for pixel-perfect alignment and artifact prevention. Debounced rendering ensures window responsiveness during resizing.
- **Recursive MRV Solver**: A deterministic, instant AI solver using the Minimum Remaining Values (MRV) heuristic.
- **Multi-Platform Support**: Linux (x86_64/ARM64), Windows (static, console-less), and Android (ARM64).
- **Smart Save/Load**: 
  - **Desktop:** Standard file dialogs with pre-filled filenames for quick persistence.
  - **Mobile (Android):** A specialized **OVERWRITE** vs. **NEW FILE** workflow to bypass native Storage Access Framework (SAF) limitations, ensuring 100% reliable file replacement and clean, unescaped filenames.
- **Raise Hand (Hint System)**: 
  - Reveals the correct value for a selected cell or the next empty cell.
  - **Pink Hinted Digits**: Hints are persistently rendered in **Bold Pink** (RGB: 255, 105, 180) to distinguish them from manual entries (White) and clues (Gold).
  - **Yellow Hint Pulse**: The hinted cell flashes in **Light Yellow** (RGB: 255, 255, 100) for 1 second upon delivery.
  - **Full Integration**: Hints trigger all collateral events (note clearing, scanning updates, and solved-checks).
- **Advanced Visual Feedback**:
  - **Conflict Highlighting**: Conflicting cells flash red for 1 second when an invalid move is attempted.
  - **Vibrant Digit Scanning**: Highlight all instances and candidates (notes) of a digit in vibrant **Light Green** (RGB: 39, 245, 63). Highlights are "sticky" across edits.
  - **Note Highlighting**: When a digit is scanned, corresponding pencil marks (notes) are also highlighted in bold light green for easier candidate tracking.
  - **Dynamic Input**: Digit buttons automatically disable when a number has been placed 9 times.
- **Dual Interaction Modes**: 
  - **SELECT Mode**: Standard navigation and cell focusing.
  - **SET Mode**: One-tap "stamping" for rapid digit entry into empty cells.
- **Power User Shortcuts**: 
  - **Tap-to-Scan**: Tapping any filled cell on the grid (val > 0) automatically triggers digit scanning for that number, regardless of the click mode.
  - **Right-Click / Long-Press**: Instantly place the highlighted digit into a cell using the current mode (**NORMAL** or **NOTES**).
  - **Keyboard Support**: Full support for 0-9, Arrows, Backspace/Delete, and **'N'** to toggle input modes.
- **Automatic Notes**: Intelligent pencil-mark management based on Sudoku rules via **AUTO NOTES**.
- **UPLOAD (Web AI Integration)**: Paste JSON strings directly from tools like ChatGPT or Gemini.

## Screenshots
### Desktop (Windows/Linux)
| Startup | Image Selection | Image Preview |
| :---: | :---: | :---: |
| ![Startup](screenshots/windows_linux_build_startup.jpg) | ![Pick Image](screenshots/windows_linux_build_pick_sudoku_puzzle_image.jpg) | ![Render Image](screenshots/windows_linux_build_render_sudoku_puzzle_image.jpg) |

| Importing | Auto Notes | Reset Board |
| :---: | :---: | :---: |
| ![Importing](screenshots/windows_linux_build_importing_sudoku_puzzle_image.jpg) | ![Auto Notes](screenshots/windows_linux_build_auto_fill_candidates_for_all_grids.jpg) | ![Reset](screenshots/windows_linux_build_reset_board_to_init.jpg) |

| Puzzle Solved |
| :---: |
| ![Solved](screenshots/windows_linux_build_puzzle_solved.jpg) |

### Android & Web AI Integration
| Android Startup | JSON Upload | Rendered Grid |
| :---: | :---: | :---: |
| ![Android Startup](screenshots/android_apk_startup.jpg) | ![JSON Upload](screenshots/android_apk_upload_json_string.jpg) | ![Rendered](screenshots/android_apk_render_uploaded_json_string.jpg) |

| AI Extraction (ChatGPT) | AI Extraction (Gemini) |
| :---: | :---: |
| ![ChatGPT](screenshots/get_9x9_grid_digits_json_string_from_chatgpt.jpg) | ![Gemini](screenshots/get_9x9_grid_digits_json_string_from_gemini.jpg) |

## Requirements
- Go 1.21+
- Fyne-cross (for multi-platform/Android builds)
- Gemini CLI v0.33.2+ (for high-accuracy image extraction with Gemini 3 models)

## Build Instructions (via WSL/Linux)
### Linux (Native/WSL)
```bash
go build -o SudokuHelper .
```

### Windows (via Fyne-cross)
```bash
fyne-cross windows -app-id com.example.sudoku_helper -arch amd64 -name SudokuHelper
```

### Android (via Fyne-cross)
```bash
fyne-cross android -app-id com.example.sudoku_helper -icon Icon.png -name SudokuHelper -arch arm64
```

## Usage
1. Launch the application.
2. **Desktop:** Use **IMPORT** to load a grid from a local screenshot (requires Gemini CLI).
3. **Web AI / Mobile:** Upload an image to ChatGPT/Gemini Web and use this prompt:
   > ACT AS A SUDOKU SCANNER. Extract the 9x9 grid from this image. Return ONLY a JSON object: {"grid": [[...],[...],...]}. Zero (0) for empty cells. CRITICAL: Every row MUST have exactly 9 numbers. NO MARKDOWN. NO CHAT.
4. Click **UPLOAD** in the app and paste the JSON string.
5. Use **SAVE** to persist your progress. 
   - On **Android**, choose **OVERWRITE** to pick an existing file to replace, or **NEW FILE** to create a fresh save with a pre-filled name.
6. Toggle input mode using the **NORMAL/NOTES** button or press **'N'**.
7. Use **Arrow Keys** to move across the board (including locked cells).
8. Use **GOLD FINGER** for an instant solution.
9. **Scan**: Tap a digit button or any number on the board to see all placements and candidates in vibrant green.
10. **Click Modes**: 
   - Use **SELECT** mode for standard navigation.
   - Use **SET** mode for rapid "one-tap" digit entry into empty cells.
11. **Shortcuts**: Use **Right-click** (or Long-press on mobile) to quickly stamp the highlighted digit into any grid.
12. **Deselect**: Click any empty area outside the grid or in the control panel to clear selection and scanning highlights.
13. **Hints**: Use the **Raise Hand** (Help Icon) for a persistent pink hint with a yellow visual pulse.

## Visual Guide
- **Startup**: Clean dark theme with a 9x9 grid and compact control rows.
- **Scanning Mode**: Grid cells and pencil marks glowing in vibrant green (39, 245, 63).
- **Conflict**: Source and destination cells flashing red (255, 0, 0).
- **Hinted Cell**: Digit rendered in pink (255, 105, 180) with a temporary yellow background (255, 255, 100).
- **Clue Cells**: Initial puzzle numbers rendered in bold gold (255, 215, 0).


## Troubleshooting
- **Right-click on Windows/Linux:** If the standard right-click is not responsive, ensure the cell is selected first or try a brief long-press.
- **Android Save (0B Files):** This issue is resolved. Use the **OVERWRITE** button when replacing existing files.
- **Windows File Picker Logs:** You may see "Error getting file attributes" in the console when browsing the `C:` root. These are harmless library logs from Fyne and do not affect functionality.
- **Gemini Extraction:** Ensure your Gemini CLI is logged in and using Gemini 3 models for the best results.

## License
MIT License
