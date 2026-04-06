package main

import (
	"fmt"

	"distancizer/internal/core"

	"fyne.io/fyne/v2"
)

func (da *DistancizerApp) calculateAll() {
	if len(da.store.POIs) == 0 || da.store.Origin == "" {
		da.setStatus("Need at least one POI and an origin address.")
		return
	}
	if da.store.OriginLat == 0 && da.store.OriginLng == 0 {
		da.setStatus("Origin address not geocoded. Set it again.")
		return
	}
	for _, poi := range da.store.POIs {
		if poi.Lat == 0 && poi.Lng == 0 {
			da.setStatus("Some POIs are not geocoded. Re-add them.")
			return
		}
	}

	origin := core.Coord{Lat: da.store.OriginLat, Lng: da.store.OriginLng}
	pois := make([]core.POI, len(da.store.POIs))
	copy(pois, da.store.POIs)
	total := len(pois)

	da.results = nil
	da.sort = sortAlpha
	da.sortSelect.Selected = "A-Z"
	da.sortSelect.Refresh()
	da.progressBar.Show()
	da.progressBar.SetValue(0)
	da.setStatus("Calculating...")

	go func() {
		for i, poi := range pois {
			result := core.CalculateOne(origin, poi)
			idx := i
			fyne.Do(func() {
				da.results = append(da.results, result)
				da.progressBar.SetValue(float64(idx+1) / float64(total))
				da.refreshResults()
			})
		}
		fyne.Do(func() {
			da.progressBar.Hide()
			da.setStatus("Calculation complete.")
		})
	}()
}

func (da *DistancizerApp) exportPOIs() {
	if len(da.store.POIs) == 0 && da.store.Origin == "" {
		da.setStatus("Nothing to export.")
		return
	}
	path, err := core.ExportPOIs(da.store)
	if err != nil {
		da.setStatus(fmt.Sprintf("Export failed: %v", err))
		return
	}
	da.setStatus(fmt.Sprintf("Exported POIs to %s", path))
}

func (da *DistancizerApp) exportResults() {
	if len(da.results) == 0 {
		da.setStatus("Nothing to export. Calculate first.")
		return
	}
	path, err := core.ExportCSV(da.store.OriginName, da.store.Origin, da.results)
	if err != nil {
		da.setStatus(fmt.Sprintf("Export failed: %v", err))
		return
	}
	da.setStatus(fmt.Sprintf("Exported to %s", path))
}
