package main

import (
	"distancizer/internal/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (da *DistancizerApp) showSetOriginDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Origin name (e.g. Home, Office)")
	nameEntry.SetText(da.store.OriginName)

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}

	dialog.NewForm("Set Origin", "Next", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			name := nameEntry.Text
			if name == "" {
				return
			}
			da.store.OriginName = name
			core.SaveStore(da.store)
			da.showOriginAddressDialog()
		},
		da.window,
	).Show()
}

func (da *DistancizerApp) showOriginAddressDialog() {
	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder("Address, Plus Code, or Google Maps URL")
	addrEntry.SetText(da.store.Origin)

	items := []*widget.FormItem{
		widget.NewFormItem("Address", addrEntry),
	}

	d := dialog.NewForm("Set Origin Address", "Search", "Cancel", items,
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
					da.store.Origin = displayAddr
					da.store.OriginLat = coord.Lat
					da.store.OriginLng = coord.Lng
					core.SaveStore(da.store)
					da.results = nil
					da.refreshOrigin()
					da.refreshResults()
					da.setStatus("Origin set.")
				},
				func(query string) {
					da.showGPSFallbackDialog(query, func(coord core.Coord, displayAddr string) {
						da.store.Origin = displayAddr
						da.store.OriginLat = coord.Lat
						da.store.OriginLng = coord.Lng
						core.SaveStore(da.store)
						da.results = nil
						da.refreshOrigin()
						da.refreshResults()
						da.setStatus("Origin set from coordinates.")
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
