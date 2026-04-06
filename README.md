# Distancizer

A commute time calculator with both a terminal UI and a macOS desktop GUI. Set an origin address and a list of Points of Interest (POIs), then calculate estimated driving times to each one using open routing data.

## Features

- **Two interfaces** — terminal TUI (Bubble Tea) and macOS desktop GUI (Fyne)
- **Address geocoding** via [Nominatim](https://nominatim.openstreetmap.org/) with interactive suggestion picker
- **Driving time calculation** via [Valhalla](https://valhalla1.openstreetmap.de/) with off-peak and estimated rush hour times (1.4x multiplier)
- **Google Plus Codes** support — full codes (e.g. `87G2GMHP+GG`) and compound codes (e.g. `MW2F+27 Wexford, PA`) are auto-detected in any address field and decoded offline
- **Google Maps URL parsing** — paste a Google Maps link containing `@lat,lng` and coordinates are extracted automatically
- **GPS coordinate fallback** — if an address can't be found, you're prompted to enter `lat, lng` or a Plus Code instead
- **Sortable results** — cycle through alphabetical, shortest-first, and longest-first sort orders
- **POIs sorted alphabetically** in the list
- **CSV export** of calculated commute times and POI/origin list
- **Persistent storage** — POIs and origin are saved to `~/.distancizer.json` between sessions (shared by both TUI and GUI)

## Building

### Terminal UI (TUI)

```
go build -o distancizer ./cmd/tui/
```

### macOS Desktop GUI

```
go build -o distancizer-gui ./gui/
```

To create a macOS `.app` bundle:

```
go install fyne.io/fyne/v2/cmd/fyne@latest
fyne package -os darwin -name Distancizer -appID com.distancizer.app -src ./gui/
```

## Usage

### TUI

```
./distancizer
```

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `a` | Add a new POI |
| `d` | Delete the selected POI |
| `o` | Set the origin address |
| `c` | Calculate commute times |
| `s` | Cycle result sort (A-Z / Shortest / Longest) |
| `e` | Export results to CSV |
| `p` | Export POIs to CSV |
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `Enter` | Confirm input or selection |
| `Esc` | Cancel current action |
| `q` / `Ctrl+C` | Quit |

### GUI

```
./distancizer-gui
```

The GUI has a top/bottom split layout with POIs on top and results on the bottom. The toolbar provides Add POI, Set Origin, and Delete actions. The results pane has Calculate, Export CSV, and sort controls. An Export POIs button in the POI pane exports all locations (origin + POIs) with name, type, and address.

### Address Input

When entering an address for a POI or origin (in either interface), you can provide any of the following:

- **Street address** — geocoded via Nominatim (e.g. `123 Main St, Pittsburgh, PA`)
- **Full Plus Code** — decoded offline (e.g. `87G2GMHP+GG`)
- **Compound Plus Code** — locality is geocoded to resolve the short code (e.g. `MW2F+27 Wexford, PA`)
- **Google Maps URL** — coordinates extracted from `@lat,lng` in the URL

If the address lookup returns no results, you'll be prompted to enter GPS coordinates (`lat, lng`) or a Plus Code as a fallback.

## Project Structure

```
distancizer/
  internal/core/    Shared business logic (geocoding, routing, store, Plus Codes, export)
  cmd/tui/          Terminal UI (Bubble Tea)
  gui/              macOS Desktop GUI (Fyne)
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components (text input, spinner)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Fyne](https://fyne.io/) — Cross-platform GUI toolkit
- [Open Location Code](https://github.com/google/open-location-code) — Plus Code encoding/decoding

## External Services

- [Nominatim](https://nominatim.openstreetmap.org/) — address geocoding and reverse geocoding
- [Valhalla](https://valhalla1.openstreetmap.de/) — driving route time calculation

## Credits

- Architected by Corey
- Created by Opus 4.6
