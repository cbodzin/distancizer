package main

import (
	"fmt"
	"strconv"
	"strings"

	"distancizer/internal/core"

	olc "github.com/google/open-location-code/go"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (da *DistancizerApp) showSuggestionPicker(results []core.GeoResult, onSelect func(core.GeoResult)) {
	items := make([]string, len(results))
	for i, r := range results {
		items[i] = r.DisplayName
	}

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("address") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(items[id])
		},
	)

	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		d.Hide()
		onSelect(results[id])
	}

	d = dialog.NewCustom("Select Address", "Cancel", list, da.window)
	d.Resize(fyne.NewSize(600, 300))
	d.Show()
}

func (da *DistancizerApp) showGPSFallbackDialog(failedQuery string, onCoord func(coord core.Coord, displayAddr string)) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("lat, lng or Plus Code (e.g. 40.6365, -80.0931 or 87G2GMHP+GG)")

	items := []*widget.FormItem{
		widget.NewFormItem("Location", entry),
	}

	d := dialog.NewForm(
		fmt.Sprintf("Address not found: %s", truncateStr(failedQuery, 50)),
		"OK", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			raw := strings.TrimSpace(entry.Text)
			if raw == "" {
				return
			}
			da.resolveGPSInput(raw, onCoord)
		},
		da.window,
	)
	d.Resize(fyne.NewSize(500, 200))
	d.Show()
}

func (da *DistancizerApp) resolveGPSInput(raw string, onCoord func(coord core.Coord, displayAddr string)) {
	inputType := core.DetectInputType(raw)

	switch inputType {
	case "full_pluscode":
		da.setStatus("Decoding Plus Code...")
		go func() {
			coord, err := core.ExtractFullPlusCode(raw)
			if err != nil {
				fyne.Do(func() { da.setStatus(fmt.Sprintf("Invalid Plus Code: %v", err)) })
				return
			}
			geo, _ := core.ReverseGeocode(coord.Lat, coord.Lng)
			displayAddr := fmt.Sprintf("%.6f, %.6f", coord.Lat, coord.Lng)
			if geo.DisplayName != "" {
				displayAddr = geo.DisplayName
			}
			fyne.Do(func() { onCoord(coord, displayAddr) })
		}()
		return

	case "compound_pluscode":
		shortCode, locality := core.ParseCompoundPlusCode(raw)
		da.setStatus("Resolving compound Plus Code...")
		go func() {
			refCoord, err := core.Geocode(locality)
			if err != nil {
				fyne.Do(func() { da.setStatus(fmt.Sprintf("Could not resolve locality: %v", err)) })
				return
			}
			fullCode, err := olc.RecoverNearest(shortCode, refCoord.Lat, refCoord.Lng)
			if err != nil {
				fyne.Do(func() { da.setStatus(fmt.Sprintf("Invalid Plus Code: %v", err)) })
				return
			}
			coord, err := core.ExtractFullPlusCode(fullCode)
			if err != nil {
				fyne.Do(func() { da.setStatus(fmt.Sprintf("Invalid Plus Code: %v", err)) })
				return
			}
			geo, _ := core.ReverseGeocode(coord.Lat, coord.Lng)
			displayAddr := fmt.Sprintf("%.6f, %.6f", coord.Lat, coord.Lng)
			if geo.DisplayName != "" {
				displayAddr = geo.DisplayName
			}
			fyne.Do(func() { onCoord(coord, displayAddr) })
		}()
		return
	}

	// Try parsing as lat, lng
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) != 2 {
		da.setStatus("Invalid format. Use: lat, lng or a Plus Code")
		return
	}
	lat, errLat := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, errLng := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if errLat != nil || errLng != nil {
		da.setStatus("Could not parse coordinates. Use: lat, lng or a Plus Code")
		return
	}

	da.setStatus("Looking up address for coordinates...")
	go func() {
		geo, _ := core.ReverseGeocode(lat, lng)
		displayAddr := fmt.Sprintf("%.6f, %.6f", lat, lng)
		if geo.DisplayName != "" {
			displayAddr = geo.DisplayName
		}
		coord := core.Coord{Lat: lat, Lng: lng}
		fyne.Do(func() { onCoord(coord, displayAddr) })
	}()
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
