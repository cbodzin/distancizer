package core

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"
)

func ExportCSV(originName string, origin string, results []CommuteResult) (string, error) {
	safe := strings.Map(func(r rune) rune {
		if r == ' ' {
			return '-'
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, originName)
	if safe == "" {
		safe = "export"
	}

	filename := fmt.Sprintf("%s-%s.csv", strings.ToLower(safe), time.Now().Format("2006-01-02"))

	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"Origin", "Destination", "Off-Peak", "Rush Hour (est.)"})

	for _, r := range results {
		offpeak := ""
		rush := ""
		if r.OK {
			offpeak = FormatMins(r.DrivingMins)
			rush = FormatMins(r.RushHourMins)
		} else if r.Error != "" {
			offpeak = r.Error
		}
		_ = w.Write([]string{origin, r.POIName, offpeak, rush})
	}

	return filename, w.Error()
}
