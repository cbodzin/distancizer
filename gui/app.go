package main

import (
	"fmt"
	"sort"
	"strings"

	"distancizer/internal/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type sortMode int

const (
	sortAlpha sortMode = iota
	sortCommuteAsc
	sortCommuteDesc
)

type DistancizerApp struct {
	window  fyne.Window
	store   core.Store
	results []core.CommuteResult
	sort    sortMode

	poiTable     *widget.Table
	originLabel  *widget.Label
	resultsTable *widget.Table
	statusLabel  *widget.Label
	progressBar  *widget.ProgressBar
	sortSelect   *widget.Select
	selectedPOI  int
}

func NewDistancizerApp(w fyne.Window) *DistancizerApp {
	store := core.LoadStore()
	sortPOIs(store.POIs)

	return &DistancizerApp{
		window:      w,
		store:       store,
		selectedPOI: -1,
	}
}

func (da *DistancizerApp) buildUI() fyne.CanvasObject {
	da.originLabel = widget.NewLabel(da.originText())
	da.originLabel.Wrapping = fyne.TextWrapWord

	da.statusLabel = widget.NewLabel("")
	da.progressBar = widget.NewProgressBar()
	da.progressBar.Hide()

	da.poiTable = widget.NewTable(
		func() (int, int) {
			return len(da.store.POIs) + 1, 2
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("placeholder text")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				headers := []string{"Name", "Address"}
				label.SetText(headers[id.Col])
				label.TextStyle.Bold = true
				return
			}
			if id.Row-1 >= len(da.store.POIs) {
				label.SetText("")
				return
			}
			poi := da.store.POIs[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(poi.Name)
			case 1:
				label.SetText(poi.Address)
			}
			label.TextStyle.Bold = false
		},
	)
	da.poiTable.SetColumnWidth(0, 150)
	da.poiTable.SetColumnWidth(1, 450)
	da.poiTable.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			da.selectedPOI = id.Row - 1
		}
	}

	da.resultsTable = widget.NewTable(
		func() (int, int) {
			return len(da.sortedResults()) + 1, 3
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("placeholder text")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				headers := []string{"Destination", "Off-Peak", "Rush Hour*"}
				label.SetText(headers[id.Col])
				label.TextStyle.Bold = true
				return
			}
			results := da.sortedResults()
			if id.Row-1 >= len(results) {
				label.SetText("")
				return
			}
			r := results[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(r.POIName)
			case 1:
				if r.OK {
					label.SetText(core.FormatMins(r.DrivingMins))
				} else {
					label.SetText(r.Error)
				}
			case 2:
				if r.OK {
					label.SetText(core.FormatMins(r.RushHourMins))
				} else {
					label.SetText("")
				}
			}
			label.TextStyle.Bold = false
		},
	)
	da.resultsTable.SetColumnWidth(0, 200)
	da.resultsTable.SetColumnWidth(1, 120)
	da.resultsTable.SetColumnWidth(2, 120)

	da.sortSelect = widget.NewSelect(
		[]string{"A-Z", "Shortest first", "Longest first"},
		func(val string) {
			switch val {
			case "A-Z":
				da.sort = sortAlpha
			case "Shortest first":
				da.sort = sortCommuteAsc
			case "Longest first":
				da.sort = sortCommuteDesc
			}
			da.refreshResults()
		},
	)
	da.sortSelect.Selected = "A-Z"

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			da.showAddPOIDialog()
		}),
		widget.NewToolbarAction(theme.HomeIcon(), func() {
			da.showSetOriginDialog()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			da.deletePOI()
		}),
	)

	exportPOIsBtn := widget.NewButtonWithIcon("Export POIs", theme.DocumentSaveIcon(), func() {
		da.exportPOIs()
	})

	poiPanel := container.NewBorder(
		container.NewHBox(widget.NewLabel("Points of Interest"), exportPOIsBtn),
		nil, nil, nil,
		da.poiTable,
	)

	calculateBtn := widget.NewButtonWithIcon("Calculate", theme.MediaPlayIcon(), func() {
		da.calculateAll()
	})
	exportBtn := widget.NewButtonWithIcon("Export CSV", theme.DocumentSaveIcon(), func() {
		da.exportResults()
	})

	resultsHeader := container.NewHBox(
		widget.NewLabel("Results"),
		da.sortSelect,
		calculateBtn,
		exportBtn,
	)

	resultsPanel := container.NewBorder(
		resultsHeader,
		nil, nil, nil,
		da.resultsTable,
	)

	content := container.NewVSplit(poiPanel, resultsPanel)
	content.SetOffset(0.4)

	statusBar := container.NewVBox(da.progressBar, da.statusLabel)

	return container.NewBorder(
		container.NewVBox(toolbar, da.originLabel),
		statusBar,
		nil, nil,
		content,
	)
}

func (da *DistancizerApp) originText() string {
	if da.store.Origin != "" {
		return fmt.Sprintf("Origin: %s - %s", da.store.OriginName, da.store.Origin)
	}
	return "Origin: (not set)"
}

func (da *DistancizerApp) refreshPOIList() {
	da.poiTable.Refresh()
}

func (da *DistancizerApp) refreshResults() {
	da.resultsTable.Refresh()
}

func (da *DistancizerApp) refreshOrigin() {
	da.originLabel.SetText(da.originText())
}

func (da *DistancizerApp) setStatus(msg string) {
	da.statusLabel.SetText(msg)
}

func (da *DistancizerApp) sortedResults() []core.CommuteResult {
	sorted := make([]core.CommuteResult, len(da.results))
	copy(sorted, da.results)
	switch da.sort {
	case sortAlpha:
		sort.Slice(sorted, func(i, j int) bool {
			return strings.ToLower(sorted[i].POIName) < strings.ToLower(sorted[j].POIName)
		})
	case sortCommuteAsc:
		sort.Slice(sorted, func(i, j int) bool {
			if !sorted[i].OK {
				return false
			}
			if !sorted[j].OK {
				return true
			}
			return sorted[i].DrivingMins < sorted[j].DrivingMins
		})
	case sortCommuteDesc:
		sort.Slice(sorted, func(i, j int) bool {
			if !sorted[i].OK {
				return true
			}
			if !sorted[j].OK {
				return false
			}
			return sorted[i].DrivingMins > sorted[j].DrivingMins
		})
	}
	return sorted
}

func sortPOIs(pois []core.POI) {
	sort.Slice(pois, func(i, j int) bool {
		return strings.ToLower(pois[i].Name) < strings.ToLower(pois[j].Name)
	})
}
