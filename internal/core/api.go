package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Coord struct {
	Lat float64
	Lng float64
}

type GeoResult struct {
	DisplayName string
	Coord       Coord
}

const rushHourMultiplier = 1.4

type CommuteResult struct {
	POIName      string
	DrivingMins  float64
	RushHourMins float64
	OK           bool
	Error        string
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func SearchAddresses(query string, limit int) ([]GeoResult, error) {
	u := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=%d&addressdetails=1",
		url.QueryEscape(query), limit)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Distancizer/1.0 (commute-calculator)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	var raw []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("bad geocoding response: %w", err)
	}

	results := make([]GeoResult, len(raw))
	for i, r := range raw {
		lat, _ := strconv.ParseFloat(r.Lat, 64)
		lng, _ := strconv.ParseFloat(r.Lon, 64)
		results[i] = GeoResult{
			DisplayName: r.DisplayName,
			Coord:       Coord{Lat: lat, Lng: lng},
		}
	}
	return results, nil
}

func Geocode(address string) (Coord, error) {
	results, err := SearchAddresses(address, 1)
	if err != nil {
		return Coord{}, err
	}
	if len(results) == 0 {
		return Coord{}, fmt.Errorf("address not found: %s", address)
	}
	return results[0].Coord, nil
}

func RouteTime(from, to Coord, costing string) (float64, error) {
	reqJSON := fmt.Sprintf(
		`{"locations":[{"lat":%f,"lon":%f},{"lat":%f,"lon":%f}],"costing":"%s"}`,
		from.Lat, from.Lng, to.Lat, to.Lng, costing,
	)
	u := "https://valhalla1.openstreetmap.de/route?json=" + url.QueryEscape(reqJSON)

	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}

		resp, err := httpClient.Get(u)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned %d", resp.StatusCode)
			continue
		}

		var result struct {
			Trip struct {
				Summary struct {
					Time float64 `json:"time"`
				} `json:"summary"`
			} `json:"trip"`
			ErrorCode int    `json:"error_code"`
			Error     string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("bad response from server")
			continue
		}
		if result.ErrorCode != 0 {
			return 0, fmt.Errorf("%s", result.Error)
		}
		return result.Trip.Summary.Time / 60.0, nil
	}
	return 0, lastErr
}

func CalculateOne(origin Coord, poi POI) CommuteResult {
	dest := Coord{Lat: poi.Lat, Lng: poi.Lng}
	r := CommuteResult{POIName: poi.Name}

	driving, err := RouteTime(origin, dest, "auto")
	if err != nil {
		r.Error = err.Error()
		return r
	}
	r.DrivingMins = driving
	r.RushHourMins = driving * rushHourMultiplier
	r.OK = true
	return r
}

func ReverseGeocode(lat, lng float64) (GeoResult, error) {
	u := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%f&lon=%f&format=json&zoom=18",
		lat, lng)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return GeoResult{}, err
	}
	req.Header.Set("User-Agent", "Distancizer/1.0 (commute-calculator)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return GeoResult{}, fmt.Errorf("reverse geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	var raw struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return GeoResult{}, fmt.Errorf("bad reverse geocoding response: %w", err)
	}
	if raw.Error != "" {
		return GeoResult{}, fmt.Errorf("reverse geocode: %s", raw.Error)
	}

	rlat, _ := strconv.ParseFloat(raw.Lat, 64)
	rlng, _ := strconv.ParseFloat(raw.Lon, 64)
	return GeoResult{
		DisplayName: raw.DisplayName,
		Coord:       Coord{Lat: rlat, Lng: rlng},
	}, nil
}

func FormatMins(mins float64) string {
	m := int(mins + 0.5)
	if m < 60 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%dh %dm", m/60, m%60)
}
