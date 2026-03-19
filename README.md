# Sudoku Helper

A Golang toolkit to extract Sudoku grids from screenshots and solve them via a feature-rich, interactive GUI.

## Features
- **AI Extraction**: Uses Gemini CLI (Gemini 3 models) to convert Sudoku screenshots into digital grids with high accuracy.
- **Interactive GUI**: Built with Fyne, featuring a responsive, touch-friendly layout with hover effects and borderless selection.
- **Advanced Visual Feedback**:
  - **Conflict Highlighting**: Conflicting cells flash red for 1 second when an invalid move is attempted.
  - **Digit Scanning**: Select a digit (1-9) to highlight all occurrences on the board in vibrant light green (RGB: 39, 245, 63). 
  - **Note Highlighting**: When a digit is scanned, corresponding pencil marks (notes) are also highlighted in bold light green for easier candidate tracking.
  - **Dynamic Input**: Digit buttons automatically disable and gray out when a number has been placed 9 times.
- **Dual Interaction Modes**: 
  - **SELECT Mode**: Left-click to focus and select a cell for standard navigation.
  - **SET Mode**: Left-click to instantly "stamp" the currently highlighted digit into an empty cell.
- **Power User Shortcuts**: 
  - **Tap-to-Scan**: Tapping any filled cell on the grid (value > 0) automatically triggers digit scanning for that number, regardless of the active click mode.
  - **Right-Click / Long-Press**: Instantly place the highlighted digit into a cell using the current input mode (**NORMAL** or **NOTES**).
- **High-Performance Grid**: Employs a 20-rectangle architecture for pixel-perfect alignment and artifact prevention. Debounced rendering ensures window responsiveness during resizing.
- **Recursive MRV Solver**: A deterministic, instant AI solver using the Minimum Remaining Values (MRV) heuristic.
- **Multi-Platform Support**: Linux (x86_64/ARM64), Windows (static, console-less), and Android (ARM64).
- **Smart Save/Load**: 
  - **Desktop:** Standard file dialogs with pre-filled filenames for quick persistence.
  - **Mobile (Android):** A specialized **OVERWRITE** vs. **NEW FILE** workflow to bypass native Storage Access Framework (SAF) limitations, ensuring 100% reliable file replacement and clean, unescaped filenames.
- **Integrated Timer**: Track your solving speed with a high-precision, pauseable playback timer.
- **Automatic Notes**: Intelligent pencil-mark management based on Sudoku rules via **AUTO NOTES**.
- **Input Modes**: Toggle between **NORMAL** and **NOTES** (pencil marks) via a dedicated button or the **'N'** key.
- **UPLOAD (Web AI Integration)**: Paste JSON strings directly from tools like ChatGPT or Gemini.
- **Fluent Navigation**: Full keyboard support (0-9, Arrows, Backspace/Delete). Navigation works even on locked clue cells.

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
9. **Digit Scanning**: Click a digit button or press 1-9 while no cell is selected to highlight all instances of that number.
10. **Tap-to-Scan**: Tapping any cell that already contains a number will automatically trigger digit scanning for that number, regardless of the active click mode.
11. **Click Modes**: 
   - Use **SELECT** mode for standard navigation.
   - Use **SET** mode for rapid "one-tap" digit entry into empty cells.
12. **Shortcuts**: Use **Right-click** (or Long-press on mobile) to quickly stamp the highlighted digit into any grid cell.
13. **Deselect**: Click any empty area outside the grid or in the control panel to clear selection and scanning highlights.

## Troubleshooting
- **Right-click on Windows/Linux:** If the standard right-click is not responsive, ensure the cell is selected first or try a brief long-press.
- **Android Save (0B Files):** This issue is resolved. Use the **OVERWRITE** button when replacing existing files.
- **Windows File Picker Logs:** You may see "Error getting file attributes" in the console when browsing the `C:` root. These are harmless library logs from Fyne and do not affect functionality.
- **Gemini Extraction:** Ensure your Gemini CLI is logged in and using Gemini 3 models for the best results.

## License
MIT License
