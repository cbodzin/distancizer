package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	olc "github.com/google/open-location-code/go"
)

var googleMapsCoordRe = regexp.MustCompile(`@(-?\d+\.?\d*),(-?\d+\.?\d*)`)

func DetectInputType(input string) string {
	trimmed := strings.TrimSpace(input)

	// Check for full Plus Code (e.g. 87G2GMHP+GG)
	if olc.CheckFull(trimmed) == nil {
		return "full_pluscode"
	}

	// Check for compound Plus Code (e.g. "MW2F+27 Wexford, PA" or "CWC8+R9, Mountain View, CA")
	// The short code is the portion up to the first space or comma after the '+'.
	if shortCode, _ := ParseCompoundPlusCode(trimmed); shortCode != "" {
		if olc.CheckShort(shortCode) == nil {
			return "compound_pluscode"
		}
	}

	// Check for Google Maps URL with @lat,lng
	lower := strings.ToLower(trimmed)
	if (strings.Contains(lower, "google.com/maps") || strings.Contains(lower, "maps.google")) &&
		googleMapsCoordRe.MatchString(trimmed) {
		return "google_maps_url"
	}

	return "address"
}

func ExtractFullPlusCode(input string) (Coord, error) {
	trimmed := strings.TrimSpace(input)
	area, err := olc.Decode(trimmed)
	if err != nil {
		return Coord{}, fmt.Errorf("invalid Plus Code: %w", err)
	}
	lat, lng := area.Center()
	return Coord{Lat: lat, Lng: lng}, nil
}

func ExtractGoogleMapsCoords(input string) (Coord, error) {
	matches := googleMapsCoordRe.FindStringSubmatch(input)
	if matches == nil {
		return Coord{}, fmt.Errorf("no coordinates found in URL")
	}
	lat, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return Coord{}, fmt.Errorf("invalid latitude in URL: %w", err)
	}
	lng, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return Coord{}, fmt.Errorf("invalid longitude in URL: %w", err)
	}
	return Coord{Lat: lat, Lng: lng}, nil
}

func ParseCompoundPlusCode(input string) (shortCode, locality string) {
	trimmed := strings.TrimSpace(input)

	// Find the '+' that's part of the Plus Code
	plusIdx := strings.Index(trimmed, "+")
	if plusIdx < 0 {
		return "", ""
	}

	// The code extends past the '+' until we hit a space, comma, or end of string
	endIdx := len(trimmed)
	for i := plusIdx + 1; i < len(trimmed); i++ {
		if trimmed[i] == ' ' || trimmed[i] == ',' {
			endIdx = i
			break
		}
	}

	shortCode = trimmed[:endIdx]
	rest := trimmed[endIdx:]
	// Strip leading comma, spaces from the locality
	locality = strings.TrimLeft(rest, ", ")
	return shortCode, locality
}
