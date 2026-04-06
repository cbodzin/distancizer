package main

import (
	"fmt"
	"time"

	"distancizer/internal/core"

	olc "github.com/google/open-location-code/go"

	"fyne.io/fyne/v2"
)

func (da *DistancizerApp) resolveAddress(input string, onSuccess func(coord core.Coord, displayAddr string), onNeedFallback func(query string), onError func(err error)) {
	inputType := core.DetectInputType(input)

	go func() {
		var coord core.Coord
		var displayAddr string

		switch inputType {
		case "full_pluscode":
			c, err := core.ExtractFullPlusCode(input)
			if err != nil {
				fyne.Do(func() { onError(err) })
				return
			}
			coord = c
			geo, err := core.ReverseGeocode(c.Lat, c.Lng)
			if err != nil {
				displayAddr = fmt.Sprintf("%.6f, %.6f", c.Lat, c.Lng)
			} else {
				displayAddr = geo.DisplayName
			}

		case "compound_pluscode":
			shortCode, locality := core.ParseCompoundPlusCode(input)
			refCoord, err := core.Geocode(locality)
			if err != nil {
				fyne.Do(func() { onError(fmt.Errorf("could not resolve locality: %w", err)) })
				return
			}
			fullCode, err := olc.RecoverNearest(shortCode, refCoord.Lat, refCoord.Lng)
			if err != nil {
				fyne.Do(func() { onError(fmt.Errorf("invalid Plus Code: %w", err)) })
				return
			}
			c, err := core.ExtractFullPlusCode(fullCode)
			if err != nil {
				fyne.Do(func() { onError(fmt.Errorf("invalid Plus Code: %w", err)) })
				return
			}
			coord = c
			geo, err := core.ReverseGeocode(c.Lat, c.Lng)
			if err != nil {
				displayAddr = fmt.Sprintf("%.6f, %.6f", c.Lat, c.Lng)
			} else {
				displayAddr = geo.DisplayName
			}

		case "google_maps_url":
			c, err := core.ExtractGoogleMapsCoords(input)
			if err != nil {
				fyne.Do(func() { onError(err) })
				return
			}
			coord = c
			geo, err := core.ReverseGeocode(c.Lat, c.Lng)
			if err != nil {
				displayAddr = fmt.Sprintf("%.6f, %.6f", c.Lat, c.Lng)
			} else {
				displayAddr = geo.DisplayName
			}

		default:
			time.Sleep(200 * time.Millisecond)
			results, err := core.SearchAddresses(input, 5)
			if err != nil {
				fyne.Do(func() { onError(err) })
				return
			}
			if len(results) == 0 {
				fyne.Do(func() { onNeedFallback(input) })
				return
			}
			if len(results) == 1 {
				coord = results[0].Coord
				displayAddr = results[0].DisplayName
			} else {
				fyne.Do(func() {
					da.showSuggestionPicker(results, func(geo core.GeoResult) {
						onSuccess(geo.Coord, geo.DisplayName)
					})
				})
				return
			}
		}

		finalCoord := coord
		finalAddr := displayAddr
		fyne.Do(func() { onSuccess(finalCoord, finalAddr) })
	}()
}
