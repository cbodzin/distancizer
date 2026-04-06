package main

import (
	"fmt"

	"distancizer/internal/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (da *DistancizerApp) showAddPOIDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("POI name (e.g. Target)")

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}

	dialog.NewForm("Add Point of Interest", "Next", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			name := nameEntry.Text
			if name == "" {
				return
			}
			da.showPOIAddressDialog(name)
		},
		da.window,
	).Show()
}

func (da *DistancizerApp) showPOIAddressDialog(poiName string) {
	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder("Address, Plus Code, or Google Maps URL")

	items := []*widget.FormItem{
		widget.NewFormItem("Address", addrEntry),
	}

	d := dialog.NewForm(fmt.Sprintf("Address for %s", poiName), "Search", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			addr := addrEntry.Text
			if addr == "" {
				return
			}
			da.setStatus("Searching for address...")
			da.resolveAddress(addr,
				func(coord core.Coord, displayAddr string) {
					da.addPOI(poiName, coord, displayAddr)
				},
				func(query string) {
					da.showGPSFallbackDialog(query, func(coord core.Coord, displayAddr string) {
						da.addPOI(poiName, coord, displayAddr)
					})
				},
				func(err error) {
					da.setStatus("Search failed: " + err.Error())
				},
			)
		},
		da.window,
	)
	d.Resize(fyne.NewSize(500, 150))
	d.Show()
}

func (da *DistancizerApp) addPOI(name string, coord core.Coord, displayAddr string) {
	poi := core.POI{
		Name:    name,
		Address: displayAddr,
		Lat:     coord.Lat,
		Lng:     coord.Lng,
	}
	da.store.POIs = append(da.store.POIs, poi)
	sortPOIs(da.store.POIs)
	core.SaveStore(da.store)
	da.results = nil
	da.refreshPOIList()
	da.refreshResults()
	da.setStatus(fmt.Sprintf("Added: %s", name))
}

func (da *DistancizerApp) deletePOI() {
	if da.selectedPOI < 0 || da.selectedPOI >= len(da.store.POIs) {
		da.setStatus("Select a POI to delete.")
		return
	}
	name := da.store.POIs[da.selectedPOI].Name
	da.store.POIs = append(da.store.POIs[:da.selectedPOI], da.store.POIs[da.selectedPOI+1:]...)
	core.SaveStore(da.store)
	da.selectedPOI = -1
	da.results = nil
	da.refreshPOIList()
	da.refreshResults()
	da.setStatus(fmt.Sprintf("Deleted: %s", name))
}
