package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type POI struct {
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat,omitempty"`
	Lng     float64 `json:"lng,omitempty"`
}

type Store struct {
	POIs       []POI   `json:"pois"`
	OriginName string  `json:"origin_name,omitempty"`
	Origin     string  `json:"origin,omitempty"`
	OriginLat  float64 `json:"origin_lat,omitempty"`
	OriginLng  float64 `json:"origin_lng,omitempty"`
}

func StorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".distancizer.json")
}

func LoadStore() Store {
	data, err := os.ReadFile(StorePath())
	if err != nil {
		return Store{}
	}
	var s Store
	_ = json.Unmarshal(data, &s)
	return s
}

func SaveStore(s Store) {
	data, _ := json.MarshalIndent(s, "", "  ")
	_ = os.WriteFile(StorePath(), data, 0644)
}
